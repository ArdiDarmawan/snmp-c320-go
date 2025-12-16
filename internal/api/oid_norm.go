package api

import "strings"

// Normalisasi supaya "1.3.6..." == ".1.3.6..."
func normOID(oid string) string {
	s := strings.TrimSpace(oid)
	s = strings.TrimPrefix(s, "SNMPv2-SMI::")
	s = strings.TrimPrefix(s, "SNMPv2-MIB::")
	if s == "" {
		return s
	}
	// pastikan selalu diawali dot
	if s[0] != '.' && s[0] >= '0' && s[0] <= '9' {
		s = "." + s
	}
	return s
}
