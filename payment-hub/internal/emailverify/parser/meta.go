package parser

type ParserMeta struct {
	ID           string
	Label        string
	SenderFilter string
	BankCode     string
}

func AllParserTypes() []ParserMeta {
	return []ParserMeta{
		{ID: "hdfc", Label: "HDFC Bank InstaAlerts", SenderFilter: "hdfcbank", BankCode: "hdfc"},
		{ID: "sbi", Label: "SBI YONO / InstaAlerts", SenderFilter: "sbi", BankCode: "sbi"},
		{ID: "icici", Label: "ICICI Bank Alerts", SenderFilter: "icicibank", BankCode: "icici"},
		{ID: "axis", Label: "Axis Bank Alerts", SenderFilter: "axisbank", BankCode: "axis"},
		{ID: "generic", Label: "Generic (any bank)", SenderFilter: "", BankCode: "generic"},
	}
}

func DefaultSenderFilter(parserType string) string {
	for _, p := range AllParserTypes() {
		if p.ID == parserType {
			return p.SenderFilter
		}
	}
	return "hdfcbank"
}
