package helpers

import (
	"context"
	"log/slog"
	"os"

	"code.cloudfoundry.org/lager/v3"
)

type textWriterSink struct {
	logger slog.Logger
}

var _ lager.Sink = &textWriterSink{}

func newTextWriterSink(logLevel lager.LogLevel) lager.Sink {
	opts := &slog.HandlerOptions{
		Level: toSlogLevel(logLevel),
	}
	slogger := slog.New(slog.NewTextHandler(os.Stdout, opts))

	return &textWriterSink {
		logger: *slogger,
	}
}

// toSlogLevel converts lager log levels to slog levels
func toSlogLevel(l lager.LogLevel) slog.Level {
	switch l {
	case lager.DEBUG:
		return slog.LevelDebug
	case lager.ERROR, lager.FATAL:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}


func (sink *textWriterSink) Log(log lager.LogFormat) {
	ctx := context.Background()
	attrs := sink.convertDataToAttrs(log)
	sink.logger.LogAttrs(ctx, toSlogLevel(log.LogLevel), log.Message, attrs...)
}

func (sink *textWriterSink) convertDataToAttrs(log lager.LogFormat) []slog.Attr {
	var attrs []slog.Attr

	if log.Source != "" {
		attrs = append(attrs, slog.String("source", log.Source))
	}

	if log.Data != nil {
		for key, value := range log.Data {
			attrs = append(attrs, slog.Any(key, value))
		}
	}

	return attrs
}
