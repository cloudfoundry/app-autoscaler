package http

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"time"

	"code.cloudfoundry.org/cli/cf/trace"
)

// TraceLoggingTransport is a thin wrapper around Transport. It dumps HTTP
// request and response using trace logger, based on the "BLUEMIX_TRACE"
// environment variable. Sensitive user data will be replaced by text
// "[PRIVATE DATA HIDDEN]".
type TraceLoggingTransport struct {
	rt     http.RoundTripper
	logger trace.Printer
}

// NewTraceLoggingTransport returns a TraceLoggingTransport wrapping around
// the passed RoundTripper. If the passed RoundTripper is nil, HTTP
// DefaultTransport is used.
func NewTraceLoggingTransport(rt http.RoundTripper, logger trace.Printer) *TraceLoggingTransport {

	if rt == nil {
		return &TraceLoggingTransport{
			rt:     http.DefaultTransport,
			logger: logger,
		}
	}
	return &TraceLoggingTransport{
		rt:     rt,
		logger: logger,
	}
}

func (r *TraceLoggingTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	start := time.Now()
	r.dumpRequest(req, start)
	resp, err = r.rt.RoundTrip(req)
	if err != nil {
		return
	}
	r.dumpResponse(resp, start)
	return
}

func (r *TraceLoggingTransport) dumpRequest(req *http.Request, start time.Time) {
	shouldDisplayBody := !strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data")

	dumpedRequest, err := httputil.DumpRequest(req, shouldDisplayBody)
	if err != nil {
		r.logger.Printf("An error occurred while dumping request:\n{{.Error}}\n", map[string]interface{}{"Error": err.Error()})
		return
	}

	r.logger.Printf("\n%s [%s]\n%s\n",
		"REQUEST:",
		start.Format(time.RFC3339),
		Sanitize(string(dumpedRequest)))

	if !shouldDisplayBody {
		r.logger.Println("[MULTIPART/FORM-DATA CONTENT HIDDEN]")
	}
}

func (r *TraceLoggingTransport) dumpResponse(res *http.Response, start time.Time) {
	end := time.Now()

	dumpedResponse, err := httputil.DumpResponse(res, true)
	if err != nil {
		r.logger.Printf("An error occurred while dumping response:\n{{.Error}}\n", map[string]interface{}{"Error": err.Error()})
		return
	}

	r.logger.Printf("\n%s [%s] %s %.0fms\n%s\n",
		"RESPONSE:",
		end.Format(time.RFC3339),
		"Elapsed:",
		end.Sub(start).Seconds()*1000,
		Sanitize(string(dumpedResponse)))
}

func Sanitize(input string) string {
	re := regexp.MustCompile(`(?m)^Authorization: .*`)
	sanitized := re.ReplaceAllString(input, "Authorization: "+PrivateDataPlaceholder())

	re = regexp.MustCompile(`password=[^&]*&`)
	sanitized = re.ReplaceAllString(sanitized, "password="+PrivateDataPlaceholder()+"&")

	sanitized = sanitizeJSON("token", sanitized)
	sanitized = sanitizeJSON("password", sanitized)

	return sanitized
}

func sanitizeJSON(propertySubstring string, json string) string {
	regex := regexp.MustCompile(fmt.Sprintf(`(?i)"([^"]*%s[^"]*)":\s*"[^\,]*"`, propertySubstring))
	return regex.ReplaceAllString(json, fmt.Sprintf(`"$1":"%s"`, PrivateDataPlaceholder()))
}

func PrivateDataPlaceholder() string {
	return "[PRIVATE DATA HIDDEN]"
}
