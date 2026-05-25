package platform

import "time"

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

func NowISO(clock Clock) string {
	return clock.Now().UTC().Format(time.RFC3339Nano)
}
