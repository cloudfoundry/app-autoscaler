// Package appstate defines the transport-independent observation contract.
// Observations are bounded-freshness hints, never authority to mutate CF.
package appstate

import (
	"context"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
)

type Observation struct {
	AppID      string    `json:"app_id"`
	Revision   int64     `json:"revision"`
	Managed    bool      `json:"managed"`
	Deleted    bool      `json:"deleted"`
	Dirty      bool      `json:"dirty"`
	State      string    `json:"state"`
	Instances  int       `json:"desired_web_instances"`
	Disabled   bool      `json:"autoscaling_disabled"`
	Routes     []string  `json:"routes"`
	VerifiedAt time.Time `json:"verified_at"`
}

type Changes struct {
	Cursor    int64         `json:"cursor"`
	CheckedAt time.Time     `json:"checked_at"`
	Items     []Observation `json:"items"`
	More      bool          `json:"more"`
	Reset     bool          `json:"reset"`
}

type Source interface {
	Changes(context.Context, int64) (Changes, error)
	Invalidate(context.Context, string) error
}

type Config struct {
	URL          string          `yaml:"url" json:"url"`
	TLS          models.TLSCerts `yaml:"tls" json:"tls"`
	PollInterval time.Duration   `yaml:"poll_interval" json:"poll_interval"`
	MaxAge       time.Duration   `yaml:"max_age" json:"max_age"`
}

func (c Config) Interval() time.Duration {
	if c.PollInterval > 0 {
		return c.PollInterval
	}
	return 5 * time.Second
}

func (c Config) Freshness() time.Duration {
	if c.MaxAge > 0 {
		return c.MaxAge
	}
	return 30 * time.Second
}
