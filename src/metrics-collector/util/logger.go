package util

import (
	"fmt"
	"metrics-collector/config"
	"os"
	"path/filepath"
	"strings"

	"github.com/pivotal-golang/lager"
)

const DEFAULT_LOG_LEVEL = lager.INFO

var Logger lager.Logger

func GetLogLevel(level string) lager.LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return lager.DEBUG
	case "INFO":
		return lager.INFO
	case "ERROR":
		return lager.ERROR
	case "FATAL":
		return lager.FATAL
	default:
		return DEFAULT_LOG_LEVEL
	}
}

func InitailizeLogger(c *config.LoggingConfig) (err error) {
	Logger = lager.NewLogger("as-metrics-collector")
	logLevel := GetLogLevel(c.Level)

	if c.LogToStdout {
		Logger.RegisterSink(lager.NewWriterSink(os.Stdout, logLevel))
	}

	if c.File != "" {
		var info os.FileInfo
		var file *os.File

		info, err = os.Stat(c.File)

		if err == nil {
			if info.IsDir() {
				err = fmt.Errorf("log file '%s' is a directory\n", c.File)
			} else {
				file, err = os.OpenFile(c.File, os.O_APPEND|os.O_RDWR, 0644)
			}
		} else {
			err = os.MkdirAll(filepath.Dir(c.File), 0744)
			if err == nil {
				file, err = os.Create(c.File)
			}
		}

		if err == nil {
			Logger.RegisterSink(lager.NewWriterSink(file, logLevel))
		}
	}
	return
}
