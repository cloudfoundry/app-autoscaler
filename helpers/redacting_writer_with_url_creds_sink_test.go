package helpers_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RedactingWriterSinkWithUrlCred", func() {
	const MaxThreads = 100

	var sink lager.Sink
	var writer *copyWriter
	var logTime time.Time

	BeforeEach(func() {
		writer = NewCopyWriter()
		logTime = time.Now()
		var err error
		sink, err = helpers.NewRedactingWriterWithURLCredSink(writer, lager.INFO, nil, nil)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when logging above the minimum log level", func() {
		BeforeEach(func() {
			sink.Log(lager.LogFormat{Timestamp: timestamp2String(logTime.UnixNano()), LogLevel: lager.INFO, Message: "hello world", Data: lager.Data{"password": "abcd", "dburl": "postgresql://username:password@hostname:5432/dbname?sslmode=disabled"}})
		})

		It("writes to the given writer", func() {
			Expect(writer.Copy()).To(MatchJSON(fmt.Sprintf(`{"message":"hello world","log_level":1,"timestamp":"%s","log_time": "%s","source":"","data":{"password":"*REDACTED*","dburl":"postgresql://username:*REDACTED*@hostname:5432/dbname?sslmode=disabled"}}`, timestamp2String(logTime.UnixNano()), time.Time.Format(logTime, time.RFC3339))))
		})
	})

	Context("when a unserializable object is passed into data", func() {
		BeforeEach(func() {
			sink.Log(lager.LogFormat{Timestamp: timestamp2String(logTime.UnixNano()), LogLevel: lager.INFO, Message: "hello world", Data: map[string]interface{}{"some_key": func() {}}})
		})

		It("logs the serialization error", func() {
			message := map[string]interface{}{}
			err := json.Unmarshal(writer.Copy(), &message)
			Expect(err).NotTo(HaveOccurred())
			Expect(message["message"]).To(Equal("hello world"))
			Expect(message["log_level"]).To(Equal(float64(1)))
			Expect(message["data"].(map[string]interface{})["lager serialisation error"]).To(Equal("json: unsupported type: func()"))
			Expect(message["data"].(map[string]interface{})["data_dump"]).ToNot(BeEmpty())
		})
	})

	Context("when logging below the minimum log level", func() {
		BeforeEach(func() {
			sink.Log(lager.LogFormat{Timestamp: timestamp2String(logTime.UnixNano()), LogLevel: lager.DEBUG, Message: "hello world"})
		})

		It("does not write to the given writer", func() {
			Expect(writer.Copy()).To(Equal([]byte{}))
		})
	})

	Context("when logging from multiple threads", func() {
		var content = "abcdefg "

		BeforeEach(func() {
			wg := new(sync.WaitGroup)
			for i := 0; i < MaxThreads; i++ {
				wg.Add(1)
				go func() {
					sink.Log(lager.LogFormat{Timestamp: timestamp2String(logTime.UnixNano()), LogLevel: lager.INFO, Message: content})
					wg.Done()
				}()
			}
			wg.Wait()
		})

		It("writes to the given writer", func() {
			lines := strings.Split(string(writer.Copy()), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				Expect(line).To(MatchJSON(fmt.Sprintf(`{"message":"%s","log_level":1,"timestamp":"%s","log_time": "%s","source":"","data":null}`, content, timestamp2String(logTime.UnixNano()), time.Time.Format(logTime, time.RFC3339))))
			}
		})
	})
})
