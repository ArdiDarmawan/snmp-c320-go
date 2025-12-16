package api

import (
	"fmt"
	"strconv"
	"strings"
)

// ambil angka ONU ID dari nama OID (ambil komponen terakhir)
func extractOnuIDFromOID(oid string) (uint32, bool) {
	oid = strings.TrimSuffix(oid, ".")
	parts := strings.Split(oid, ".")
	if len(parts) == 0 {
		return 0, false
	}
	last := parts[len(parts)-1]
	u, err := strconv.ParseUint(last, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(u), true
}

// normalisasi string/[]byte jadi string
func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return ""
	}
}

// serial ZTE kadang "1,SN" => ambil SN
func normalizeSerial(v any) string {
	s := strings.TrimSpace(toString(v))
	if s == "" {
		return ""
	}
	// versi repo: kalau prefix "1," langsung buang
	if strings.HasPrefix(s, "1,") {
		return strings.TrimSpace(s[2:])
	}
	// jaga-jaga kalau format "onuId,SN"
	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		last := strings.TrimSpace(parts[len(parts)-1])
		if last != "" {
			return last
		}
	}
	return s
}

// konversi value numeric SNMP jadi int64 aman (support int/int64/uint/uint32/dll)
func toInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int:
		return int64(x), true
	case int32:
		return int64(x), true
	case int64:
		return x, true
	case uint:
		return int64(x), true
	case uint32:
		return int64(x), true
	case uint64:
		// hati-hati overflow, tapi biasanya kecil
		if x > uint64(^uint32(0)) {
			return 0, false
		}
		return int64(x), true
	default:
		return 0, false
	}
}

// RX dBm versi repo: value * 0.002 - 30
// return string 2 desimal
func convertRxDbm(v any) (string, error) {
	n, ok := toInt64(v)
	if !ok {
		return "", fmt.Errorf("value is not integer-like")
	}
	result := float64(n)*0.002 - 30.0
	return strconv.FormatFloat(result, 'f', 2, 64), nil
}

func convertDbmFromScaledInt(v any) (float64, bool) {
	n, ok := toInt64(v)
	if !ok {
		return 0, false
	}
	// dari contoh: -23205 -> -23.205 dBm (skala 1000)
	return float64(n) / 1000.0, true
}

func statusTextFromCode(v any) string {
	n, ok := toInt64(v)
	if !ok {
		return "Unknown"
	}
	switch n {
	case 1:
		return "Logging"
	case 2:
		return "LOS"
	case 3:
		return "Synchronization"
	case 4:
		return "Online"
	case 5:
		return "Dying Gasp"
	case 6:
		return "Auth Failed"
	case 7:
		return "Offline"
	default:
		return "Unknown"
	}
}

func offlineReasonTextFromCode(v any) string {
	n, ok := toInt64(v)
	if !ok {
		return "Unknown"
	}
	switch n {
	case 1:
		return "Unknown"
	case 2:
		return "LOS"
	case 3:
		return "LOSi"
	case 4:
		return "LOFi"
	case 5:
		return "sfi"
	case 6:
		return "loai"
	case 7:
		return "loami"
	case 8:
		return "AuthFail"
	case 9:
		return "PowerOff"
	case 10:
		return "deactiveSucc"
	case 11:
		return "deactiveFail"
	case 12:
		return "Reboot"
	case 13:
		return "Shutdown"
	default:
		return "Unknown"
	}
}
