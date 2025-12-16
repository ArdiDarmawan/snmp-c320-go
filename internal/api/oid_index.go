package api

import (
	"strconv"
	"strings"
)

// ambil angka terakhir dari OID string
// contoh: "1.3.6....1.7.4" -> 4
func oidLastIndex(oid string) int {
	parts := strings.Split(oid, ".")
	if len(parts) == 0 {
		return 0
	}
	last := parts[len(parts)-1]
	n, err := strconv.Atoi(last)
	if err != nil {
		return 0
	}
	return n
}
