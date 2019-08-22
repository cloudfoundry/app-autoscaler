package helpers

import (
	"code.cloudfoundry.org/lager"
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
	l.logger.Debug(message, lager.Data{"data": data})
}
func (l *LoggregatorGRPCLogger) Panicf(message string, data ...interface{}) {
	l.logger.Fatal(message, nil, lager.Data{"data": data})
}
