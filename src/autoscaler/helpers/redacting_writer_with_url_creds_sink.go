package helpers

import (
	"io"
	"sync"

	"code.cloudfoundry.org/lager"
)

type redactingWriterWithURLCredSink struct {
	writer                  io.Writer
	minLogLevel             lager.LogLevel
	writeL                  *sync.Mutex
	jsonRedacterWithURLCred *JSONRedacterWithURLCred
}

func NewRedactingWriterWithURLCredSink(writer io.Writer, minLogLevel lager.LogLevel, keyPatterns []string, valuePatterns []string) (lager.Sink, error) {
	jsonRedacterWithURLCred, err := NewJSONRedacterWithURLCred(keyPatterns, valuePatterns)
	if err != nil {
		return nil, err
	}
	return &redactingWriterWithURLCredSink{
		writer:                  writer,
		minLogLevel:             minLogLevel,
		writeL:                  new(sync.Mutex),
		jsonRedacterWithURLCred: jsonRedacterWithURLCred,
	}, nil
}

func (sink *redactingWriterWithURLCredSink) Log(log lager.LogFormat) {
	if log.LogLevel < sink.minLogLevel {
		return
	}
	timeLogFormat := NewTimeLogFormat(log)
	sink.writeL.Lock()
	v := timeLogFormat.ToJSON()
	rv := sink.jsonRedacterWithURLCred.Redact(v)
	sink.writer.Write(rv)
	sink.writer.Write([]byte("\n"))
	sink.writeL.Unlock()
}
