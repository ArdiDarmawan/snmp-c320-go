package api

import (
	"encoding/hex"
	"time"
)

// RFC2579 DateAndTime: 8 bytes atau 11 bytes
// 8:  Y(2) M D H M S DS
// 11: + dir('+'/'-') TZH TZM
func decodeDateAndTimeBytes(b []byte) (time.Time, bool) {
	if len(b) != 8 && len(b) != 11 {
		return time.Time{}, false
	}

	year := int(b[0])<<8 | int(b[1])
	month := int(b[2])
	day := int(b[3])
	hour := int(b[4])
	min := int(b[5])
	sec := int(b[6])
	deci := int(b[7]) // 1/10 detik

	ns := deci * 100_000_000
	loc := time.Local

	if len(b) == 11 {
		dir := b[8] // '+' atau '-'
		tzh := int(b[9])
		tzm := int(b[10])
		offset := tzh*3600 + tzm*60
		if dir == '-' {
			offset = -offset
		}
		loc = time.FixedZone("SNMP", offset)
	}

	t := time.Date(year, time.Month(month), day, hour, min, sec, ns, loc)
	return t, true
}

func bytesHex(b []byte) string {
	return hex.EncodeToString(b)
}
