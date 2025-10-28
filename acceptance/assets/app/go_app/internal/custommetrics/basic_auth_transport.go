package api

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
)

type BasicAuthTransport struct {
	Username  string
	Password  string
	Transport *http.Transport
}

func (bat BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s",
		base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s",
			bat.Username, bat.Password)))))
	return bat.Transport.RoundTrip(req)
}

func (bat *BasicAuthTransport) Client() *http.Client {
	return &http.Client{Transport: bat}
}

func NewBasicAuthTransport(credentials CustomMetricsCredentials) *BasicAuthTransport {
	//#nosec G402 -- test app that shall run on dev foundations without proper certs
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &BasicAuthTransport{credentials.Username, credentials.Password, tr}
}
