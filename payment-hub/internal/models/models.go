package models

import "time"

const (
	OrderStatusPending    = "pending"
	OrderStatusProcessing = "processing"
	OrderStatusSuccess    = "success"
	OrderStatusFailed     = "failed"
	OrderStatusExpired    = "expired"
	OrderStatusRefunded   = "refunded"

	MerchantStatusActive    = "active"
	MerchantStatusSuspended = "suspended"
)

type Merchant struct {
	ID         string
	Name       string
	Domain     string
	APIKey     string
	APISecret  string
	WebhookURL        string
	ReturnURL         string
	Status            string
	PaymentProfileID  string
	CreatedAt         time.Time
	UpdatedAt  time.Time
}

type Order struct {
	ID                 string
	HubOrderID         string
	MerchantID         string
	MerchantOrderID    string
	PaymentToken       string
	Amount             float64
	PayAmount          float64
	Currency           string
	PaymentProvider    string
	PaymentProfileID   string
	Status             string
	CustomerEmail      string
	CustomerName       string
	CustomerPhone      string
	ProductName        string
	ProductDescription string
	ReturnURL          string
	WebhookURL         string
	PhonePeTxnID       string
	CustomerUTR        string
	PaidAt             *time.Time
	ExpiresAt          time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreateOrderInput struct {
	OrderID     string
	Amount      float64
	Currency    string
	Customer    CustomerInput
	Product     ProductInput
	ReturnURL   string
	WebhookURL  string
}

type CustomerInput struct {
	Email string
	Name  string
	Phone string
}

type ProductInput struct {
	Name        string
	Description string
}

type CreateOrderResult struct {
	HubOrderID string
	PaymentURL string
	ExpiresAt  time.Time
}

type VerifyOrderResult struct {
	HubOrderID   string
	OrderID      string
	Status       string
	Amount       float64
	Currency     string
	PhonePeTxnID string
	PaidAt       *time.Time
}

type PaymentProfile struct {
	ID           string
	Name         string
	UPIID        string
	PayeeName    string
	BankCode     string
	IMAPHost     string
	IMAPPort     int
	IMAPUser     string
	IMAPPassword string
	SenderFilter string
	ParserType   string
	IsActive     bool
	IMAPLastOKAt      *time.Time
	IMAPLastError     string
	IMAPLastCheckedAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AdminUser struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
}

type MerchantUser struct {
	ID             string
	Email          string
	PasswordHash   string
	Name           string
	MerchantID     string
	OnboardingDone bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
