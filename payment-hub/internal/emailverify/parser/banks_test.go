package parser

import "testing"

func TestParseSBI(t *testing.T) {
	body := "Dear Customer, Rs. 250.00 credited to A/c **1234 via UPI. UPI Ref No: 523456789012. SBI"
	alert, ok := ParseSBI(body)
	if !ok {
		t.Fatal("expected parse success")
	}
	if alert.Amount != 250.00 {
		t.Fatalf("amount=%v", alert.Amount)
	}
	if alert.UTR != "523456789012" {
		t.Fatalf("utr=%s", alert.UTR)
	}
}

func TestParseGeneric(t *testing.T) {
	body := "INR 99.50 received. UTR: 412345678901"
	alert, ok := ParseGeneric(body)
	if !ok {
		t.Fatal("expected parse success")
	}
	if alert.Amount != 99.50 || alert.UTR != "412345678901" {
		t.Fatalf("got amount=%v utr=%s", alert.Amount, alert.UTR)
	}
}
