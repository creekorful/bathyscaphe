package clock

//go:generate mockgen -destination=../clock_mock/client_mock.go -package=clock_mock . Clock

import "time"

type Clock interface {
	Now() time.Time
}

type SystemClock struct {
}

func (clock *SystemClock) Now() time.Time {
	return time.Now()
}
