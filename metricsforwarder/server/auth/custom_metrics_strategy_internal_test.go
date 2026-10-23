package auth

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics submission strategies", func() {
	Context("with the same-app strategy", func() {
		It("accepts the app's own metrics without consulting a binding", func() {
			strategy := &DefaultMetricsSubmissionStrategy{}
			Expect(strategy.validate("app", "app", nil, nil, nil)).To(Succeed())
		})

		It("rejects metrics for another app", func() {
			strategy := &DefaultMetricsSubmissionStrategy{}
			Expect(strategy.validate("target", "submitter", nil, nil, nil)).To(MatchError(ErrorAppIDWrong))
		})
	})

	Context("with the bound-app strategy", func() {
		It("accepts the app's own metrics without consulting a binding", func() {
			strategy := &BoundedMetricsSubmissionStrategy{}
			Expect(strategy.validate("app", "app", nil, nil, nil)).To(Succeed())
		})
	})
})
