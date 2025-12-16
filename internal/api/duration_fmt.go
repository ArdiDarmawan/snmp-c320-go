package api

import (
	"fmt"
	"time"
)

func formatDurationDHMS(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	sec := int64(d.Seconds())

	days := sec / 86400
	sec %= 86400
	hours := sec / 3600
	sec %= 3600
	mins := sec / 60
	sec %= 60

	return fmt.Sprintf("%d days %d hours %d minutes %d seconds", days, hours, mins, sec)
}
