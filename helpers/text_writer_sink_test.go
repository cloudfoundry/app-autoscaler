package helpers_test

import (
	"bytes"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TextWriterSink", func() {
	const MaxThreads = 100

	var sink lager.Sink
	var logTime time.Time
	buffer := bytes.NewBuffer([]byte{})

	BeforeEach(func() {
		logTime = time.Now()
		sink = helpers.NewTextWriterSink(GinkgoWriter, lager.INFO)
		buffer.Reset()
	})

	Context("Logging level", func() {
		When("logging above the minimum log level", func() {
			BeforeEach(func() {
				GinkgoWriter.TeeTo(buffer)
				sink.Log(lager.LogFormat{Timestamp: timestamp2String(logTime.UnixNano()), LogLevel: lager.INFO, Message: "hello world", Data: lager.Data{"password": "abcd", "dburl": "postgresql://username:password@hostname:5432/dbname?sslmode=disabled"}})
			})

			It("writes to the writer", func() {
				output := buffer.String()

				Expect(output).To(ContainSubstring("hello world"))
				Expect(output).To(ContainSubstring("password=abcd"))
				Expect(output).To(ContainSubstring("dburl=\"postgresql://username:password@hostname:5432/dbname?sslmode=disabled\""))
			})
		})

		When("logging below the minimum log level", func() {
			BeforeEach(func() {
				GinkgoWriter.TeeTo(buffer)
				sink.Log(lager.LogFormat{Timestamp: timestamp2String(logTime.UnixNano()), LogLevel: lager.DEBUG, Message: "hello world"})
			})

			It("does not write to the writer", func() {
				output := buffer.String()
				Expect(output).To(BeEmpty())
			})
		})
	})
	Context("when logging from multiple threads", func() {
		var content = "abcdefg "

		BeforeEach(func() {
			GinkgoWriter.TeeTo(buffer)
			wg := new(sync.WaitGroup)
			for range MaxThreads {
				wg.Add(1)
				go func() {
					sink.Log(lager.LogFormat{Timestamp: timestamp2String(logTime.UnixNano()), LogLevel: lager.INFO, Message: content})
					wg.Done()
				}()
			}
			wg.Wait()
		})

		It("writes to stdout", func() {
			output := buffer.String()
			lines := strings.Split(output, "\n")
			count := 0
			for _, line := range lines {
				if strings.Contains(line, content) {
					count++
				}
			}
			Expect(count).To(Equal(MaxThreads))
		})
	})
	Context("when logging with complex data", func() {
		BeforeEach(func() {
			GinkgoWriter.TeeTo(buffer)
			sink.Log(lager.LogFormat{
				Timestamp: timestamp2String(logTime.UnixNano()),
				LogLevel:  lager.INFO,
				Message:   "complex data",
				Source:    "test-source",
				Data: lager.Data{
					"string":  "value",
					"number":  42,
					"boolean": true,
					"nested": map[string]interface{}{
						"key": "value",
					},
				},
			})
		})

		It("handles complex data structures", func() {
			output := buffer.String()

			Expect(output).To(ContainSubstring("complex data"))
			Expect(output).To(ContainSubstring("source=test-source"))
			Expect(output).To(ContainSubstring("string=value"))
			Expect(output).To(ContainSubstring("number=42"))
			Expect(output).To(ContainSubstring("boolean=true"))
		})
	})

	Context("when logging errors", func() {
		BeforeEach(func() {
			GinkgoWriter.TeeTo(buffer)
			sink.Log(lager.LogFormat{
				Timestamp: timestamp2String(logTime.UnixNano()),
				LogLevel:  lager.ERROR,
				Message:   "error occurred",
				Data: lager.Data{
					"error": "something went wrong",
				},
			})
		})

		It("logs error messages", func() {
			output := buffer.String()

			Expect(output).To(ContainSubstring("error occurred"))
			Expect(output).To(ContainSubstring("error=\"something went wrong\""))
		})
	})
})
