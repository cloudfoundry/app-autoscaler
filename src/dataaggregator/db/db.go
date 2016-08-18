package db

import (
	"dataaggregator/policy"
)

type DB interface {
	RetrievePolicies() ([]*policy.PolicyJson, error)
	Close() error
}
