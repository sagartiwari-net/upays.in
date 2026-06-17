package parser

import (
	"regexp"
	"strconv"
)

var (
	genericAmountRe = regexp.MustCompile(`(?i)(?:Rs\.?|INR)\s*([0-9]+(?:\.[0-9]{1,2})?)`)
	genericUTRPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:UPI\s*)?(?:Ref(?:erence)?\.?|reference)\s*(?:No\.?|number)?\s*:?\s*(\d{10,})`),
		regexp.MustCompile(`(?i)UTR[:\s]+(\d{10,})`),
		regexp.MustCompile(`(?i)transaction\s*(?:id|ref)[:\s]+(\d{10,})`),
	}
)

func ParseGeneric(body string) (*CreditAlert, bool) {
	amountMatch := genericAmountRe.FindStringSubmatch(body)
	if amountMatch == nil {
		return nil, false
	}
	var utr string
	for _, re := range genericUTRPatterns {
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
