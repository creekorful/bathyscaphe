package clock

//go:generate mockgen -destination=../clock_mock/client_mock.go -package=clock_mock . Clock

import "time"

// Clock is an interface to ease unit testing
type Clock interface {
	// Now return current time
	Now() time.Time
}

// SystemClock is a clock that use system time
type SystemClock struct {
}

// Now return now from system clock
func (clock *SystemClock) Now() time.Time {
	return time.Now()
}
