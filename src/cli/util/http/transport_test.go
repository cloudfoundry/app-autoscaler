package http_test

import (
	"net/http"
	"strings"
	"time"

	. "cli/util/http"

	"code.cloudfoundry.org/cli/cf/trace"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Transport Test", func() {

	var (
		tServer *ghttp.Server
		client  *http.Client
		url     string
		buffer  *gbytes.Buffer
		logger  trace.Printer
	)

	BeforeEach(func() {
		tServer = ghttp.NewServer()
		tServer.RouteToHandler("GET", "/hello",
			ghttp.RespondWithJSONEncoded(http.StatusOK, "welcome"),
		)
		url = tServer.URL() + "/hello"
	})

	AfterEach(func() {
		tServer.Close()
	})

	Context("Dump request/response when trace Enabled", func() {

		BeforeEach(func() {
			buffer = gbytes.NewBuffer()
			logger = trace.NewLogger(buffer, false, "true", "")
			tr := NewTraceLoggingTransport(&http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
			}, logger)

			client = &http.Client{Transport: tr}
		})

		It("It dump request and response", func() {
			req, _ := http.NewRequest("GET", url, nil)
			client.Do(req)

			Expect(buffer).To(gbytes.Say("REQUEST:.*"))
			Expect(buffer).To(gbytes.Say("GET /hello HTTP/1.1"))
			Expect(buffer).To(gbytes.Say("Host: " + strings.TrimPrefix(tServer.URL(), "http://")))
			Expect(buffer).To(gbytes.Say("RESPONSE:"))
			Expect(buffer).To(gbytes.Say("HTTP/1.1 200 OK"))
			Expect(buffer).To(gbytes.Say("Content-Type:"))
			Expect(buffer).To(gbytes.Say("Date"))
			Expect(buffer).To(gbytes.Say("welcome"))

		})

		It("It dump request and response, but sanitize auth info", func() {

			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Add("Authorization", "Bear xxxxxx")
			client.Do(req)

			Expect(buffer).To(gbytes.Say("REQUEST:.*"))
			Expect(buffer).To(gbytes.Say("GET /hello HTTP/1.1"))
			Expect(buffer).To(gbytes.Say("Host: " + strings.TrimPrefix(tServer.URL(), "http://")))
			Expect(buffer).To(gbytes.Say("Authorization:" + PrivateDataPlaceholder()))

			Expect(buffer).To(gbytes.Say("RESPONSE:"))
			Expect(buffer).To(gbytes.Say("HTTP/1.1 200 OK"))
			Expect(buffer).To(gbytes.Say("Content-Type:"))
			Expect(buffer).To(gbytes.Say("Date"))
			Expect(buffer).To(gbytes.Say("welcome"))
		})

	})

	Context("No request/response dump when trace disabled", func() {

		BeforeEach(func() {
			buffer = gbytes.NewBuffer()
			logger = trace.NewLogger(buffer, false, "false", "")
			tr := NewTraceLoggingTransport(&http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
			}, logger)

			client = &http.Client{Transport: tr}
		})

		It("No request and response dump", func() {
			req, _ := http.NewRequest("GET", url, nil)
			client.Do(req)

			Expect(buffer).ToNot(gbytes.Say("REQUEST:.*"))
			Expect(buffer).ToNot(gbytes.Say("GET /hello HTTP/1.1"))
			Expect(buffer).ToNot(gbytes.Say("Host: " + strings.TrimPrefix(tServer.URL(), "http://")))
			Expect(buffer).ToNot(gbytes.Say("RESPONSE:"))
			Expect(buffer).ToNot(gbytes.Say("HTTP/1.1 200 OK"))
			Expect(buffer).ToNot(gbytes.Say("Content-Type:"))
			Expect(buffer).ToNot(gbytes.Say("Date"))
			Expect(buffer).ToNot(gbytes.Say("welcome"))

		})

	})

})
