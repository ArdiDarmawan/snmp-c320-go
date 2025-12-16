package snmp

import "strings"

// JoinBaseRel(".1.3.6.1.4.1.3902.1082", ".10.10...") => ".1.3.6.1.4.1.3902.1082.10.10..."
func JoinBaseRel(base, rel string) string {
	base = strings.TrimSuffix(base, ".")
	rel = strings.TrimPrefix(rel, ".")
	return base + "." + rel
}

// JoinIndexes(oid, 285278721, 3) => "<oid>.285278721.3"
func JoinIndexes(oid string, idx ...uint32) string {
	oid = strings.TrimSuffix(oid, ".")
	for _, v := range idx {
		oid += "." + itoaU32(v)
	}
	return oid
}

func itoaU32(v uint32) string {
	// tiny fast conversion without fmt (kept simple)
	if v == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	return string(buf[i:])
}
