package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/assets/amember"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/emailverify"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/emailverify/parser"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/services"
	"github.com/gofiber/fiber/v2"
)

func (h *AdminHandler) TestProfileIMAP(c *fiber.Ctx) error {
	p, err := h.profiles.Get(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	imap := emailverify.NewIMAPClient(p.IMAPHost, p.IMAPPort, p.IMAPUser, p.IMAPPassword, p.SenderFilter)
	result, err := imap.TestConnection(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if result.OK {
		_ = h.profileRepo.UpdateIMAPHealth(c.Context(), p.ID, true, "")
	} else {
		_ = h.profileRepo.UpdateIMAPHealth(c.Context(), p.ID, false, result.Message)
	}
	return c.JSON(fiber.Map{
		"ok":       result.OK,
		"message":  result.Message,
		"subjects": result.Subjects,
	})
}

func (h *AdminHandler) TestProfileParse(c *fiber.Ctx) error {
	p, err := h.profiles.Get(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var body struct {
		EmailBody string `json:"email_body"`
		FetchLatest bool `json:"fetch_latest"`
	}
	_ = c.BodyParser(&body)

	emailBody := strings.TrimSpace(body.EmailBody)
	if body.FetchLatest || emailBody == "" {
		imap := emailverify.NewIMAPClient(p.IMAPHost, p.IMAPPort, p.IMAPUser, p.IMAPPassword, p.SenderFilter)
		fetched, err := imap.FetchLatestBody(c.Context())
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "fetch latest failed: " + err.Error()})
		}
		emailBody = fetched
	}

	alert, ok := parser.Parse(p.ParserType, emailBody)
	if !ok {
		return c.JSON(fiber.Map{
			"matched": false,
			"message": "Could not extract amount + UTR from email. Check parser type or paste full email body.",
		})
	}
	return c.JSON(fiber.Map{
		"matched": true,
		"amount":  alert.Amount,
		"utr":     alert.UTR,
	})
}

func (h *AdminHandler) TriggerProfilePoll(c *fiber.Ctx) error {
	if h.emailWorker == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email worker not enabled"})
	}
	if _, err := h.profiles.Get(c.Context(), c.Params("id")); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	h.emailWorker.TriggerPoll()
	return c.JSON(fiber.Map{"ok": true, "message": "Poll triggered — check transactions in ~30 seconds"})
}

func (h *AdminHandler) ListUnmatched(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	items, total, err := h.bankTxns.ListUnmatched(c.Context(), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list unmatched"})
	}
	out := make([]fiber.Map, 0, len(items))
	for _, item := range items {
		out = append(out, fiber.Map{
			"id":           item.ID,
			"utr":          item.UTR,
			"amount":       item.Amount,
			"profile_id":   item.ProfileID,
			"profile_name": item.ProfileName,
			"raw_excerpt":  item.RawExcerpt,
			"created_at":   item.CreatedAt,
		})
	}
	return c.JSON(fiber.Map{"unmatched": out, "total": total})
}

func (h *AdminHandler) ManualApproveOrder(c *fiber.Ctx) error {
	var body struct {
		UTR      string  `json:"utr"`
		Amount   float64 `json:"amount"`
		BankTxnID string `json:"bank_txn_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	body.UTR = strings.TrimSpace(body.UTR)
	if body.UTR == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "utr required"})
	}

	order, err := h.orders.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if order.Status != models.OrderStatusPending && order.Status != models.OrderStatusProcessing {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "order is not pending"})
	}
	if body.Amount > 0 && !repository.AmountsEqual(body.Amount, order.PayAmount) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "amount mismatch — order expects pay_amount " + strconv.FormatFloat(order.PayAmount, 'f', 2, 64),
		})
	}

	exists, err := h.bankTxns.UTRExists(c.Context(), body.UTR)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "UTR already used"})
	}

	if err := h.orders.MarkSuccess(c.Context(), order.ID, body.UTR, nil); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if body.BankTxnID != "" {
		_ = h.bankTxns.LinkOrder(c.Context(), body.BankTxnID, order.ID)
	} else {
		_ = h.bankTxns.Record(c.Context(), body.UTR, "manual-"+order.ID, order.PayAmount, order.ID, order.PaymentProfileID, "manual approve")
	}

	updated, err := h.orders.GetByHubOrderID(c.Context(), order.HubOrderID)
	if err == nil && h.notifier != nil {
		h.notifier.NotifyAsync(updated, "payment.success")
	}
	return c.JSON(fiber.Map{"ok": true, "hub_order_id": order.HubOrderID, "status": "success"})
}

func (h *AdminHandler) ExportOrdersCSV(c *fiber.Ctx) error {
	items, _, err := h.orders.ListAdmin(c.Context(), repository.OrderListFilter{
		Status:     c.Query("status"),
		MerchantID: c.Query("merchant_id"),
		Search:     strings.TrimSpace(c.Query("q")),
		Limit:      5000,
		Offset:     0,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "export failed"})
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"hub_order_id", "merchant_order_id", "merchant", "domain", "amount", "pay_amount", "status", "customer_email", "utr", "created_at"})
	for _, o := range items {
		_ = w.Write([]string{
			o.HubOrderID, o.MerchantOrderID, o.MerchantName, o.MerchantDomain,
			strconv.FormatFloat(o.Amount, 'f', 2, 64),
			strconv.FormatFloat(o.PayAmount, 'f', 2, 64),
			o.Status, o.CustomerEmail, o.CustomerUTR, o.CreatedAt,
		})
	}
	w.Flush()

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", `attachment; filename="transactions.csv"`)
	return c.Send(buf.Bytes())
}

func (h *AdminHandler) DownloadAmemberPlugin(c *fiber.Ctx) error {
	content := amember.Plugin
	if len(content) == 0 {
		paths := []string{
			filepath.Join("..", "payment-hub-sdk-php", "amember-plugin", "upipays.php"),
			filepath.Join("payment-hub-sdk-php", "amember-plugin", "upipays.php"),
		}
		for _, p := range paths {
			if b, err := os.ReadFile(p); err == nil {
				content = b
				break
			}
		}
	}
	if len(content) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "plugin file not found"})
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("upipays.php")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "zip failed"})
	}
	if _, err := fw.Write(content); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "zip failed"})
	}
	_ = zw.Close()

	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", `attachment; filename="upipays-amember-plugin.zip"`)
	return c.Send(buf.Bytes())
}

func (h *AdminHandler) MerchantRevenue(c *fiber.Ctx) error {
	days, _ := strconv.Atoi(c.Query("days", "30"))
	rows, err := h.orders.MerchantRevenue(c.Context(), days)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load revenue"})
	}
	out := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		out = append(out, fiber.Map{
			"merchant_id":   r.MerchantID,
			"merchant_name": r.MerchantName,
			"domain":        r.Domain,
			"orders":        r.Orders,
			"revenue":       r.Revenue,
		})
	}
	return c.JSON(fiber.Map{"merchants": out, "days": days})
}

func (h *AdminHandler) IMAPAlerts(c *fiber.Ctx) error {
	since := time.Now().UTC().Add(-1 * time.Hour)
	profiles, err := h.profileRepo.ListIMAPAlerts(c.Context(), since)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load alerts"})
	}
	out := make([]fiber.Map, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, fiber.Map{
			"id":                   p.ID,
			"name":                 p.Name,
			"imap_user":            p.IMAPUser,
			"imap_last_ok_at":      services.ProfileResponse(p)["imap_last_ok_at"],
			"imap_last_error":      p.IMAPLastError,
			"imap_last_checked_at": services.ProfileResponse(p)["imap_last_checked_at"],
		})
	}
	return c.JSON(fiber.Map{"alerts": out})
}
