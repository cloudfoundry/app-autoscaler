package helpers_test

import (
	"encoding/json"
	"fmt"
	"time"

	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/lager/v3"
)

var _ = Describe("TimeLoggerFormat", func() {

	var log lager.LogFormat
	var logTime time.Time
	var template = `{"timestamp":"%s","source":"log-source","message":"message","log_level":1,"data":{"log-data":"log-data"},"log_time":"%s"}`
	var tlf TimeLogFormat

	JustBeforeEach(func() {
		tlf = NewTimeLogFormat(log)
	})

	Context("when all values of LogFormat are valid", func() {
		BeforeEach(func() {
			logTime = time.Now()
			log = lager.LogFormat{
				Timestamp: timestamp2String(logTime.UnixNano()),
				Source:    "log-source",
				Message:   "message",
				LogLevel:  lager.INFO,
				Data:      map[string]interface{}{"log-data": "log-data"},
			}
		})
		It("add RFC3339 format field log_time by the value LogFormat.Timestamp to output", func() {
			result := fmt.Sprintf(template, timestamp2String(logTime.UnixNano()), time.Time.Format(logTime, time.RFC3339))
			Expect(tlf.ToJSON()).To(MatchJSON(result))
		})
	})

	Context("when there is no LogFormat.Timestamp", func() {
		BeforeEach(func() {
			logTime = time.Now()
			log = lager.LogFormat{
				Source:   "log-source",
				Message:  "message",
				LogLevel: lager.INFO,
				Data:     map[string]interface{}{"log-data": "log-data"},
			}
		})
		It("add RFC3339 format field log_time by the value 0 to output", func() {
			result := fmt.Sprintf(template, "", time.Time.Format(time.Unix(0, 0), time.RFC3339))
			Expect(tlf.ToJSON()).To(MatchJSON(result))
		})
	})

	Context("when the value of LogFormat.Timestamp is an invalid float", func() {
		BeforeEach(func() {
			logTime = time.Now()
			log = lager.LogFormat{
				Timestamp: "not-an-invalid-float-string",
				Source:    "log-source",
				Message:   "message",
				LogLevel:  lager.INFO,
				Data:      map[string]interface{}{"log-data": "log-data"},
			}
		})
		It("add RFC3339 format field log_time by the value 0 to output", func() {
			result := fmt.Sprintf(template, "not-an-invalid-float-string", time.Time.Format(time.Unix(0, 0), time.RFC3339))
			Expect(tlf.ToJSON()).To(MatchJSON(result))
		})
	})

	Context("when a unserializable object is passed into LogFormat.Data", func() {
		BeforeEach(func() {
			logTime = time.Now()
			log = lager.LogFormat{
				Timestamp: timestamp2String(logTime.UnixNano()),
				Source:    "log-source",
				Message:   "message",
				LogLevel:  lager.INFO,
				Data:      map[string]interface{}{"log-data": func() {}},
			}
		})
		It("logs the serialization error", func() {
			message := map[string]interface{}{}
			err := json.Unmarshal(tlf.ToJSON(), &message)
			Expect(err).ToNot(HaveOccurred())
			Expect(message["message"]).To(Equal("message"))
			Expect(message["timestamp"]).To(Equal(timestamp2String(logTime.UnixNano())))
			Expect(message["log_time"]).To(Equal(time.Time.Format(logTime, time.RFC3339)))
			Expect(message["log_level"]).To(Equal(float64(1)))
			Expect(message["data"].(map[string]interface{})["lager serialisation error"]).To(Equal("json: unsupported type: func()"))
			Expect(message["data"].(map[string]interface{})["data_dump"]).ToNot(BeEmpty())
		})
	})
})
