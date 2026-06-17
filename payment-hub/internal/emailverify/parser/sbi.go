package parser

import (
	"regexp"
	"strconv"
)

var (
	sbiAmountRe = regexp.MustCompile(`(?i)(?:Rs\.?|INR)\s*([0-9]+(?:\.[0-9]{1,2})?)`)
	sbiUTRPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)UPI Ref(?:erence)?\.?\s*(?:No\.?)?\s*:?\s*(\d{10,})`),
		regexp.MustCompile(`(?i)Ref(?:erence)?\.?\s*(?:No\.?)?\s*:?\s*(\d{10,})`),
		regexp.MustCompile(`(?i)UTR[:\s]+(\d{10,})`),
	}
)

func ParseSBI(body string) (*CreditAlert, bool) {
	amountMatch := sbiAmountRe.FindStringSubmatch(body)
	if amountMatch == nil {
		return nil, false
	}
	var utr string
	for _, re := range sbiUTRPatterns {
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
	return &CreditAlert{Amount: amount, UTR: utr}, true
}
