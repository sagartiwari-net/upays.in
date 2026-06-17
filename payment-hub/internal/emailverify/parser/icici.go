package parser

import (
	"regexp"
	"strconv"
)

var (
	iciciAmountRe = regexp.MustCompile(`(?i)(?:Rs\.?|INR)\s*([0-9]+(?:\.[0-9]{1,2})?)`)
	iciciUTRPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)UPI\s*(?:transaction\s*)?(?:reference|ref)\s*(?:no\.?|number)?\s*:?\s*(\d{10,})`),
		regexp.MustCompile(`(?i)UTR[:\s]+(\d{10,})`),
		regexp.MustCompile(`(?i)Ref(?:erence)?\.?\s*(?:No\.?)?\s*:?\s*(\d{10,})`),
	}
)

func ParseICICI(body string) (*CreditAlert, bool) {
	amountMatch := iciciAmountRe.FindStringSubmatch(body)
	if amountMatch == nil {
		return nil, false
	}
	var utr string
	for _, re := range iciciUTRPatterns {
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
