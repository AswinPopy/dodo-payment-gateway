package webhook

import "time"

const (
	// MaxAttempts is the delivery budget, including the first try.
	MaxAttempts = 8

	// AttemptTimeout is the HTTP timeout for a single delivery.
	AttemptTimeout = 5 * time.Second

	// DispatcherPollInterval is how often the worker looks for due events.
	DispatcherPollInterval = 1 * time.Second

	// ClaimBatchSize is how many due events one poll claims.
	ClaimBatchSize = 10
)

// BackoffAfterFailure is the wait after attempt N fails, before attempt N+1.
// After attempt 8 fails the event is dead-lettered (FAILED).
//
//	attempt 1 (immediate) → +5s
//	attempt 2             → +25s
//	attempt 3             → +2m
//	attempt 4             → +10m
//	attempt 5             → +30m
//	attempt 6             → +2h
//	attempt 7             → +6h
//	attempt 8             → FAILED
//
// Total budget from first try: 8h 42m 30s.
var BackoffAfterFailure = []time.Duration{
	5 * time.Second,
	25 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	6 * time.Hour,
}

func NextRetryAt(failedAttempts int, now time.Time) (time.Time, bool) {
	if failedAttempts >= MaxAttempts {
		return time.Time{}, false
	}
	if failedAttempts <= 0 {
		return now, true
	}
	if failedAttempts > len(BackoffAfterFailure) {
		return time.Time{}, false
	}
	return now.Add(BackoffAfterFailure[failedAttempts-1]), true
}
