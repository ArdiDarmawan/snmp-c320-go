package api

func scaleMilli(v any) (float64, bool) {
	n, ok := toInt64(v)
	if !ok {
		return 0, false
	}
	// raw -1000 berarti N/A (khusus current)
	return float64(n) * 0.001, true
}
