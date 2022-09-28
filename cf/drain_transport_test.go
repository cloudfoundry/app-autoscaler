package cf_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"
	"github.com/stretchr/testify/assert"
)

type FakeReadCloser struct {
	io.Reader
	io.Closer

	numBytes       int
	closeWasCalled bool
}

func (f *FakeReadCloser) Read(p []byte) (int, error) {
	n, err := f.Reader.Read(p)

	f.numBytes += n

	return n, err
}

func (f *FakeReadCloser) Close() error {
	f.closeWasCalled = true
	return nil
}

var _ io.ReadCloser = &FakeReadCloser{}

type testRoundTripper struct {
	req *http.Request
	res *http.Response
	err error
}

func (t *testRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	t.req = request
	return t.res, t.err
}

var _ http.RoundTripper = &testRoundTripper{}

func TestRoundTripper(t *testing.T) {
	tripResp := &http.Response{}
	testT := &testRoundTripper{res: tripResp}

	transport := cf.DrainingTransport{Transport: testT}
	req, _ := http.NewRequest("GET", "some-url", nil)
	//nolint:bodyclose
	resp, _ := transport.RoundTrip(req)

	assert.Equal(t, testT.req, req)
	assert.Equal(t, resp, tripResp)
}

func TestTripBodyClose(t *testing.T) {
	tripResp := &http.Response{}
	req, _ := http.NewRequest("GET", "some-url", nil)
	data := "some buffer data"
	body := FakeReadCloser{Reader: strings.NewReader(data)}
	tripResp.Body = &body
	testT := &testRoundTripper{res: tripResp}
	transport := cf.DrainingTransport{Transport: testT}
	resp, _ := transport.RoundTrip(req)
	_ = resp.Body.Close()
	assert.Equal(t, body.numBytes, len(data))
	assert.True(t, body.closeWasCalled)
}

func TestRoundTripperError(t *testing.T) {
	testT := &testRoundTripper{res: nil, err: errors.New("some-error")}

	transport := cf.DrainingTransport{Transport: testT}
	req, _ := http.NewRequest("GET", "some-url", nil)
	//nolint:bodyclose
	resp, err := transport.RoundTrip(req)

	assert.Equal(t, err.Error(), "some-error")
	assert.Nil(t, resp)
}
