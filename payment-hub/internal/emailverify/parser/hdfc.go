package parser

import (
	"regexp"
	"strconv"
)

var (
	hdfcAmountRe = regexp.MustCompile(`Rs\.?\s*([0-9]+\.[0-9]{2})`)
	hdfcUTRPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)UPI Reference No\.?:\s*(\d+)`),
		regexp.MustCompile(`(?i)reference number is\s*(\d+)`),
	}
)

type CreditAlert struct {
	Amount float64
	UTR    string
}

func ParseHDFC(body string) (*CreditAlert, bool) {
	amountMatch := hdfcAmountRe.FindStringSubmatch(body)
	if amountMatch == nil {
		return nil, false
	}

	var utr string
	for _, re := range hdfcUTRPatterns {
		if m := re.FindStringSubmatch(body); m != nil {
			utr = m[1]
			break
		}
	}
	if utr == "" {
		return nil, false
	}

	amount, err := strconv.ParseFloat(amountMatch[1], 64)
	if err != nil {
		return nil, false
	}

	return &CreditAlert{
		Amount: amount,
		UTR:    utr,
	}, true
}
