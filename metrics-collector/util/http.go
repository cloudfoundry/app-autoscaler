package util

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	hutil "net/http/httputil"
	"regexp"
	"strings"
)

type errorReponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func CreateJsonErrorResponse(c string, m string) []byte {
	body := &errorReponse{
		Code:    c,
		Message: m,
	}
	bytes, _ := json.Marshal(body)
	return bytes
}

var transport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

var httpClient *http.Client = &http.Client{Transport: transport}

func Sanitize(input string) (sanitized string) {
	var sanitizeJson = func(propertyName string, json string) string {
		regex := regexp.MustCompile(fmt.Sprintf(`"%s":\s*"[^"]*"`, propertyName))
		return regex.ReplaceAllString(json, fmt.Sprintf(`"%s":"%s"`, propertyName, PRIVATE_DATA_PLACEHOLDER()))
	}

	re := regexp.MustCompile(`(?m)^Authorization: .*`)
	sanitized = re.ReplaceAllString(input, "Authorization: "+PRIVATE_DATA_PLACEHOLDER())
	re = regexp.MustCompile(`password=[^&]*&`)
	sanitized = re.ReplaceAllString(sanitized, "password="+PRIVATE_DATA_PLACEHOLDER()+"&")

	sanitized = sanitizeJson("access_token", sanitized)
	sanitized = sanitizeJson("refresh_token", sanitized)
	sanitized = sanitizeJson("token", sanitized)
	sanitized = sanitizeJson("password", sanitized)

	return
}

func PRIVATE_DATA_PLACEHOLDER() string {
	return "[PRIVATE DATA HIDDEN]"
}

func DumpRequest(req *http.Request) (message string) {
	shouldDisplayBody := !strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data")

	dumpedRequest, err := hutil.DumpRequest(req, shouldDisplayBody)
	if err != nil {
		Logger.Error("Dump http request", err)
		return
	}

	message = fmt.Sprintf("%s\n", Sanitize(string(dumpedRequest)))
	if !shouldDisplayBody {
		message += "[MULTIPART/FORM-DATA CONTENT HIDDEN]\n"
	}
	return
}

func DumpResponse(res *http.Response) (message string) {
	dumpedResponse, err := hutil.DumpResponse(res, true)
	if err != nil {
		Logger.Error("Dump http response", err)
		return
	}

	message = fmt.Sprintf("%s\n", Sanitize(string(dumpedResponse)))
	return
}

func DoRequest(method, url, token string, headers map[string]string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return
	}

	if token != "" {
		req.Header.Set("Authorization", token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	Logger.Debug("Dump http", map[string]interface{}{"Request": DumpRequest(req)})

	resp, err = httpClient.Do(req)
	if err == nil {
		Logger.Debug("Dump http", map[string]interface{}{"Response": DumpResponse(resp)})
	}
	return
}
