package util

import "time"

func TimeToNumber(date time.Time) int64 {
	return date.UnixNano() / int64(time.Millisecond)
}
