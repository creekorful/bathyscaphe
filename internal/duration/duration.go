package duration

import (
	"github.com/xhit/go-str2duration/v2"
	"time"
)

// ParseDuration parse given duration into time.Duration
// or returns -1 if fails
func ParseDuration(duration string) time.Duration {
	if duration == "" {
		return -1
	}

	val, err := str2duration.ParseDuration(duration)
	if err != nil {
		return -1
	}

	return val
}
