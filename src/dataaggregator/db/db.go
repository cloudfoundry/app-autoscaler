package db

import (
	"dataaggregator/appmetric"
	"dataaggregator/policy"
)

type DB interface {
	RetrievePolicies() ([]*policy.PolicyJson, error)
	SaveAppMetric(appMetric *appmetric.AppMetric) error
	Close() error
}
