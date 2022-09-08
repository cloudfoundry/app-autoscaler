package cf

import (
	"io"
	"net/http"
)

type TripBody struct {
	io.ReadCloser
}

func (t TripBody) Close() error {
	_, _ = io.Copy(io.Discard, t.ReadCloser)

	return t.ReadCloser.Close()
}

var _ io.ReadCloser = TripBody{}

var _ http.RoundTripper = DrainingTransport{}

type DrainingTransport struct {
	Transport http.RoundTripper
}

func (d DrainingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	resp, err := d.Transport.RoundTrip(request)

	if err != nil {
		return resp, err
	}
	resp.Body = TripBody{resp.Body}
	return resp, nil
}
