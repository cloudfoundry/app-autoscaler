package helpers

import (
	"fmt"

	"code.cloudfoundry.org/lager/v3"
)

type LoggregatorGRPCLogger struct {
	logger lager.Logger
}

func NewLoggregatorGRPCLogger(logger lager.Logger) *LoggregatorGRPCLogger {
	return &LoggregatorGRPCLogger{
		logger: logger,
	}
}
func (l *LoggregatorGRPCLogger) Printf(message string, data ...interface{}) {
	l.logger.Info(fmt.Sprint(message, data), lager.Data{"data": data})
}
func (l *LoggregatorGRPCLogger) Panicf(message string, data ...interface{}) {
	l.logger.Fatal(fmt.Sprintf(message, data...), nil, lager.Data{"data": data})
}
