package brokerserver_test

import (
	"errors"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/api/brokerserver"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BrokerError", func() {
	Context("When a brokereError.Error() is called and has an err wrapped", func() {
		It("has the correct error string", func() {
			err2 := errors.New("some error message")
			var err error = &brokerserver.BrokerError{
				Status:  404,
				Message: "Some message",
				Err:     fmt.Errorf("wrapping err: %w", err2),
				Data:    lager.Data{"some": "Data"},
			}
			Expect(errors.Is(err, err2)).To(BeTrue())
			Expect(err.Error()).To(Equal("Some message, statusCode(404): wrapping err: some error message"))
		})
	})
	Context("When a brokereError.Error() is called with nil error", func() {
		It("has the correct error string", func() {
			err2 := errors.New("some error message")
			err := &brokerserver.BrokerError{}
			Expect(errors.Is(err, err2)).To(BeFalse())
			Expect(err.Unwrap()).To(BeNil())
			Expect(err.Error()).To(Equal("uninitialised, statusCode(0)"))
		})
	})
})
