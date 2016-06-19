package mhttp

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func CreateJsonErrorResponse(code string, msg string) []byte {
	body := &ErrorResponse{
		Code:    code,
		Message: msg,
	}
	bytes, _ := json.Marshal(body)
	return bytes
}

var transport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

var httpClient = &http.Client{Transport: transport}

func DoRequest(method, url, token string, headers map[string]string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return httpClient.Do(req)
}
