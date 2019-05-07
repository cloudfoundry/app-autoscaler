package helpers

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/lager"
)

type LoggingConfig struct {
	Level string `yaml:"level"`
}

func InitLoggerFromConfig(conf *LoggingConfig, name string) lager.Logger {
	logLevel, err := getLogLevel(conf.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %s\n", err.Error())
		os.Exit(1)
	}
	logger := lager.NewLogger(name)

	keyPatterns := []string{"[Pp]wd", "[Pp]ass", "[Ss]ecret", "[Tt]oken"}

	redactedSink, err := NewRedactingWriterWithURLCredSink(os.Stdout, logLevel, keyPatterns, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create redacted sink: %s\n", err.Error())
		os.Exit(1)
	}
	logger.RegisterSink(redactedSink)

	return logger
}

func getLogLevel(level string) (lager.LogLevel, error) {
	switch level {
	case "debug":
		return lager.DEBUG, nil
	case "info":
		return lager.INFO, nil
	case "error":
		return lager.ERROR, nil
	case "fatal":
		return lager.FATAL, nil
	default:
		return -1, fmt.Errorf("Error: unsupported log level:%s", level)
	}
}
