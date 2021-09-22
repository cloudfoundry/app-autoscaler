package server

import (
	"fmt"
)

var (
	ErrorNoPolicy          = fmt.Errorf("no policy defined")
	ErrorStdMetricExists   = fmt.Errorf("metric already exists in std metrics")
	ErrorMetricNotInPolicy = fmt.Errorf("metric is not define in the policy")
)

var _ MetricError = &Error{}

type Error struct {
	metric string
	err    error
}

func (e *Error) Unwrap() error {
	return e.err
}

func (e *Error) Error() string {
	return fmt.Sprintf("Metric Error:%s - %s", e.metric, e.err)
}

func (e *Error) GetMetricName() string {
	return e.metric
}

type MetricError interface {
	GetMetricName() string
	Unwrap() error
	Error() string
}
