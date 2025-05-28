package testhelpers

import (
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
