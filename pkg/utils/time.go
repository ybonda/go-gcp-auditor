// pkg/utils/time.go
package utils

import "time"

// TimeRange represents a time period with start and end times
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// NewTimeRange creates a new TimeRange for the last n days
func NewTimeRange(days int) TimeRange {
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -days)
	return TimeRange{
		Start: start,
		End:   end,
	}
}

// Duration returns the duration of the time range
func (tr TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}
