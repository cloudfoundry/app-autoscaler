package cf

import (
	"code.cloudfoundry.org/lager/v3"
	"github.com/hashicorp/go-retryablehttp"
)

var _ retryablehttp.LeveledLogger = &LeveledLoggerAdapter{}

type LeveledLoggerAdapter struct{ lager.Logger }

func (l LeveledLoggerAdapter) Error(msg string, keysAndValues ...interface{}) {
	l.Logger.Error(msg, nil, createData(keysAndValues))
}

func (l LeveledLoggerAdapter) Info(msg string, keysAndValues ...interface{}) {
	l.Logger.Info(msg, createData(keysAndValues))
}

func (l LeveledLoggerAdapter) Debug(msg string, keysAndValues ...interface{}) {
	l.Logger.Debug(msg, createData(keysAndValues))
}

func (l LeveledLoggerAdapter) Warn(msg string, keysAndValues ...interface{}) {
	//This is because lager.logger does not have a warning level ... need to replace it.
	l.Logger.Info("Warning: "+msg, createData(keysAndValues))
}

func createData(keysAndValues []interface{}) lager.Data {
	data := lager.Data{}

	for i := 0; (i + 1) < len(keysAndValues); i += 2 {
		key, isString := keysAndValues[i].(string)
		if isString {
			data[key] = keysAndValues[i+1]
		}
	}
	return data
}
