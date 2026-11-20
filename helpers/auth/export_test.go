package auth

import "time"

// Only tests can control the cache clock; production callers cannot bypass refresh.
func (v *ChainVerifier) SetClockForTest(now func() time.Time) {
	v.now = now
}
