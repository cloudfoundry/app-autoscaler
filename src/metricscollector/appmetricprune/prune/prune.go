package prune

import (
	"db"
	"time"

	"code.cloudfoundry.org/lager"
)

const TokenTypeBearer = "bearer"

type Prune struct {
	logger   lager.Logger
	metricDB db.MetricsDB
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewPrune(logger lager.Logger, metricDB db.MetricsDB) *Prune {
	return &Prune{
		logger:   logger,
		metricDB: metricDB,
	}
}

func (pr *Prune) PruneMetricsOlderThan(cutoffDays int) {
	timestamp := time.Now().AddDate(0, 0, -cutoffDays).UnixNano()

	pr.metricDB.PruneMetrics(timestamp)
}
