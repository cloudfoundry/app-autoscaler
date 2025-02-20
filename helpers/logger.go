package helpers

import (
	"fmt"
	"log/slog"
	"os"

	"code.cloudfoundry.org/lager/v3"
)

type LoggingConfig struct {
	Level         string `yaml:"level" json:"level"`
	PlainTextSink bool   `yaml:"plaintext_sink" json:"plaintext_sink"`
}

func InitLoggerFromConfig(conf *LoggingConfig, name string) lager.Logger {
	logLevel, err := parseLogLevel(conf.Level)
	if err != nil {
		handleError("failed to initialize logger", err)
	}

	logger := lager.NewLogger(name)

	if conf.PlainTextSink {
		plaintextFormatSink := createPlaintextSink()
		logger.RegisterSink(plaintextFormatSink)
	} else {
		redactedSink := createRedactedSink(logLevel)
		logger.RegisterSink(redactedSink)
	}

	return logger
}

func parseLogLevel(level string) (lager.LogLevel, error) {
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
		return -1, fmt.Errorf("unsupported log level: %s", level)
	}
}

func createPlaintextSink() lager.Sink {
	slogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return lager.NewSlogSink(slogger)
}

func createRedactedSink(logLevel lager.LogLevel) lager.Sink {
	keyPatterns := []string{"[Pp]wd", "[Pp]ass", "[Ss]ecret", "[Tt]oken"}
	redactedSink, err := NewRedactingWriterWithURLCredSink(os.Stdout, logLevel, keyPatterns, nil)
	if err != nil {
		handleError("failed to create redacted sink", err)
	}
	return redactedSink
}

func handleError(message string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", message, err.Error())
	os.Exit(1)
}
