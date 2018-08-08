package models

import (
	"time"
)

type Lock struct {
	Owner                 string
	LastModifiedTimestamp time.Time
	Ttl                   time.Duration
}
