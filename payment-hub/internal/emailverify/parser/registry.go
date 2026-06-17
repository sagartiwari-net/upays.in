package parser

import "strings"

func Parse(parserType, body string) (*CreditAlert, bool) {
	switch strings.ToLower(strings.TrimSpace(parserType)) {
	case "sbi":
		return ParseSBI(body)
	case "icici":
		return ParseICICI(body)
	case "axis":
		return ParseAxis(body)
	case "generic":
		return ParseGeneric(body)
	case "hdfc", "":
		return ParseHDFC(body)
	default:
		return ParseGeneric(body)
	}
}
