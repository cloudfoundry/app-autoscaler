package prune_test

import (
	"time"

	. "metricscollector/appmetricprune/prune"
	"metricscollector/fakes"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Prune", func() {
	var (
		metricDB        *fakes.FakeMetricsDB
		logger          lager.Logger
		prune           *Prune
		timeBeforePrune time.Time
		timeAfterPrune  time.Time
		cutoffDays      int
	)

	BeforeEach(func() {

		logger = lager.NewLogger("prune-test")

		metricDB = &fakes.FakeMetricsDB{}

	})
	Describe("when a request for prune old metric data comes", func() {

		Context("prune metric data", func() {
			BeforeEach(func() {
				cutoffDays = 20

				prune = NewPrune(logger, metricDB)
			})

			JustBeforeEach(func() {
				timeBeforePrune = time.Now()
				prune.PruneMetricsOlderThan(cutoffDays)
				timeAfterPrune = time.Now()
			})

			It("It has the the correct timestamp cutoff for pruning", func() {
				Expect(metricDB.PruneMetricsCallCount()).To(Equal(1))

				Expect(metricDB.PruneMetricsArgsForCall(0) >= timeBeforePrune.AddDate(0, 0, -cutoffDays).UnixNano()).To(Equal(true))
				Expect(metricDB.PruneMetricsArgsForCall(0) <= timeAfterPrune.AddDate(0, 0, -cutoffDays).UnixNano()).To(Equal(true))
			})

		})

	})
})
