package helpers

import (
	"context"
	"io"
	"log/slog"
	"time"

	"code.cloudfoundry.org/lager/v3"
)

type textWriterSink struct {
	logger slog.Logger
}

var _ lager.Sink = &textWriterSink{}

func NewTextWriterSink(writer io.Writer, logLevel lager.LogLevel) lager.Sink {
	opts := &slog.HandlerOptions{
		Level: toSlogLevel(logLevel),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Don't display nanoseconds – as for our non slog-based log-entries.
			if a.Key == slog.TimeKey {
				return slog.String("time", a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}
	slogger := slog.New(slog.NewTextHandler(writer, opts))

	return &textWriterSink{
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
	// In theory we would prefer to use the approach taken upstream in
	// <https://github.com/cloudfoundry/lager/blob/d157756475eda86f343d254c199652a48261a2a6/slog_sink.go#L28-L36>.
	// However we don't have access to the internal time. Instead we accept here a small delay by
	// letting slog again measure the time on its own.
	//
	// Even preferable would be to just create a slog_sink for lager-logger. However they currently
	// have a bug in their code preventing them to check, if the minimum log-level is met by a
	// logged message.

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
