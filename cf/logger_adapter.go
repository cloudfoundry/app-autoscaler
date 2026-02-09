package cf

import "code.cloudfoundry.org/lager/v3"

// LeveledLoggerAdapter adapts lager.Logger to retryablehttp's LeveledLogger interface
type LeveledLoggerAdapter struct {
	lager.Logger
}

func (l LeveledLoggerAdapter) Error(msg string, keyval ...interface{}) {
	l.Logger.Error(msg, nil, toData(keyval))
}

func (l LeveledLoggerAdapter) Warn(msg string, keyval ...interface{}) {
	l.Logger.Info(msg, toData(keyval))
}

func (l LeveledLoggerAdapter) Info(msg string, keyval ...interface{}) {
	l.Logger.Info(msg, toData(keyval))
}

func (l LeveledLoggerAdapter) Debug(msg string, keyval ...interface{}) {
	l.Logger.Debug(msg, toData(keyval))
}

func toData(keyval []interface{}) lager.Data {
	data := lager.Data{}
	for i := 0; i < len(keyval)-1; i += 2 {
		if key, ok := keyval[i].(string); ok {
			data[key] = keyval[i+1]
		}
	}
	return data
}
