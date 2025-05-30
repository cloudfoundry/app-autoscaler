package testhelpers

import (
	"io"
	"net/http"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"
)

func CheckHealthAuth(t GinkgoTInterface, client *http.Client, url string, username, password string, expectedStatus int) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	Expect(err).NotTo(HaveOccurred())

	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	Expect(err).NotTo(HaveOccurred())

	defer resp.Body.Close()
	Expect(resp.StatusCode).To(Equal(expectedStatus))
}

func CheckHealthResponse(client *http.Client, url string, expected []string) {
	rsp, err := client.Get(url)
	Expect(err).NotTo(HaveOccurred())
	Expect(rsp.StatusCode).To(Equal(http.StatusOK))
	raw, _ := io.ReadAll(rsp.Body)
	healthData := string(raw)
	for _, s := range expected {
		Expect(healthData).To(ContainSubstring(s))
	}
	rsp.Body.Close()
}
