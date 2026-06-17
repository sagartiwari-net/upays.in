package parser

import "testing"

func TestParseHDFC_LegacyFormat(t *testing.T) {
	body := "Dear Customer, Rs. 160.00 is successfully credited to your account **7722 by VPA karanraj2804@ybl Mr Karan Raj on 18-04-26. Your UPI transaction reference number is 620132678501. Thank you for banking with us. Warm Regards, HDFC Bank"
	alert, ok := ParseHDFC(body)
	if !ok {
		t.Fatal("expected parse success")
	}
	if alert.Amount != 160.00 {
		t.Fatalf("amount=%v", alert.Amount)
	}
	if alert.UTR != "620132678501" {
		t.Fatalf("utr=%s", alert.UTR)
	}
}

func TestParseHDFC_InstaAlertFormat(t *testing.T) {
	body := `Dear Customer,

Greetings from HDFC Bank!

We're writing to inform you that Rs.1.00 has been successfully credited to your HDFC Bank account ending in 7722.

Transaction Details:
a. Date: 13-06-26
b. Sender: Pilla Sai Likith (VPA: likith@superyes)
c. UPI Reference No.: 653070886774

Thank you for banking with HDFC Bank.`
	alert, ok := ParseHDFC(body)
	if !ok {
		t.Fatal("expected parse success")
	}
	if alert.Amount != 1.00 {
		t.Fatalf("amount=%v", alert.Amount)
	}
	if alert.UTR != "653070886774" {
		t.Fatalf("utr=%s", alert.UTR)
	}
}
