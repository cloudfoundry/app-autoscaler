package app_test

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/app"
	"code.cloudfoundry.org/app-autoscaler-release/src/acceptance/assets/app/go_app/internal/app/appfakes"
	"github.com/fgrosse/zaptest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/steinfletcher/apitest"
)

var _ = Describe("Disk handler", func() {

	var mockDiskOccupier *appfakes.FakeDiskOccupier

	apiTest := func(diskOccupier app.DiskOccupier) *apitest.APITest {
		GinkgoHelper()
		logger := zaptest.LoggerWriter(GinkgoWriter)

		return apitest.New().Handler(app.Router(logger, nil, nil, nil, diskOccupier, nil))
	}

	BeforeEach(func() {
		mockDiskOccupier = &appfakes.FakeDiskOccupier{}
	})

	It("should err if utilization not an int64", func() {
		apiTest(mockDiskOccupier).
			Get("/disk/invalid/4").
			Expect(GinkgoT()).
			Status(http.StatusBadRequest).
			Body(`{"error":{"description":"invalid utilization: strconv.ParseInt: parsing \"invalid\": invalid syntax"}}`).
			End()
	})
	It("should err if disk out of bounds", func() {
		apiTest(mockDiskOccupier).
			Get("/disk/100001010101010249032897287298719874687936483275648273632429479827398798271/4").
			Expect(GinkgoT()).
			Status(http.StatusBadRequest).
			Body(`{"error":{"description":"invalid utilization: strconv.ParseInt: parsing \"100001010101010249032897287298719874687936483275648273632429479827398798271\": value out of range"}}`).
			End()
	})
	It("should err if disk not an int", func() {
		apiTest(mockDiskOccupier).
			Get("/disk/5/invalid").
			Expect(GinkgoT()).
			Status(http.StatusBadRequest).
			Body(`{"error":{"description":"invalid minutes: strconv.ParseInt: parsing \"invalid\": invalid syntax"}}`).
			End()
	})
	It("should return ok", func() {
		apiTest(mockDiskOccupier).
			Get("/disk/100/2").
			Expect(GinkgoT()).
			Status(http.StatusOK).
			Body(`{"utilization":100, "minutes":2 }`).
			End()
	})
	It("should err if already running", func() {
		mockDiskOccupier.OccupyReturns(errors.New("already occupying"))
		apiTest(mockDiskOccupier).
			Get("/disk/100/2").
			Expect(GinkgoT()).
			Status(http.StatusInternalServerError).
			Body(`{"error":{"description":"error invoking occupation: already occupying"}}`).
			End()
	})
})

var _ = Describe("DefaultDiskOccupier", func() {

	var diskOccupier app.DiskOccupier

	var filePath string

	const duration = 2 * time.Second
	const veryShortTime = 10 * time.Millisecond
	const oneHundredKB = 100 * 1000 // 100 KB

	BeforeEach(func() {
		filePath = filepath.Join(GinkgoT().TempDir(), "this-file-is-being-used-to-eat-up-the-disk")
		diskOccupier = app.NewDefaultDiskOccupier(filePath)
	})

	Describe("Occupy", func() {
		When("not occupying already", func() {
			It("occupies oneHundredKB for a certain amount of time", func() {
				err := diskOccupier.Occupy(oneHundredKB, duration)
				Expect(err).ToNot(HaveOccurred())

				fStat, err := os.Stat(filePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(fStat.Size()).To(Equal(int64(oneHundredKB)))

				Eventually(func() bool {
					return isGone(filePath)
				}, 2*duration, 50*time.Millisecond)
			})
		})

		When("occupying already started", func() {
			BeforeEach(func() {
				err := diskOccupier.Occupy(oneHundredKB, duration)
				Expect(err).ToNot(HaveOccurred())
			})

			It("fails with an error", func() {
				err := diskOccupier.Occupy(oneHundredKB, duration)
				Expect(err).To(MatchError(errors.New("disk space is already being occupied")))
			})
		})

		When("occupation just ended", func() {
			BeforeEach(func() {
				err := diskOccupier.Occupy(oneHundredKB, veryShortTime)
				Expect(err).ToNot(HaveOccurred())

				// wait till occupation is over
				time.Sleep(2 * veryShortTime)
			})

			It("is possible to start occupy again", func() {
				err := diskOccupier.Occupy(oneHundredKB, veryShortTime)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("occupation was stopped", func() {
			BeforeEach(func() {
				err := diskOccupier.Occupy(oneHundredKB, duration)
				Expect(err).ToNot(HaveOccurred())

				diskOccupier.Stop()
			})

			It("is possible to start occupy again", func() {
				err := diskOccupier.Occupy(oneHundredKB, duration)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("Stop", func() {
		When("occupying already", func() {
			BeforeEach(func() {
				tremendousAmountOfTime := 999999999 * duration
				err := diskOccupier.Occupy(oneHundredKB, tremendousAmountOfTime)
				Expect(err).ToNot(HaveOccurred())
			})

			It("stops occupying oneHundredKB", func() {
				diskOccupier.Stop()

				Expect(isGone(filePath))
			})
		})

		When("not occupying already", func() {
			It("does nothing", func() {
				diskOccupier.Stop()

				Expect(true)
			})
		})

		When("occupation just ended", func() {
			BeforeEach(func() {

				err := diskOccupier.Occupy(oneHundredKB, veryShortTime)
				Expect(err).ToNot(HaveOccurred())

				// wait till occupation is over
				time.Sleep(2 * veryShortTime)
			})

			It("does nothing", func() {
				diskOccupier.Stop()

				Expect(true)
			})
		})

		When("someone bangs the head against the API", func() {
			headbangs := 1000
			maxOccupyDuration := 5 * time.Millisecond
			maxWait := 5 * time.Millisecond

			const Occupy = 1
			const Stop = 2
			OccupyOrStop := func() int {
				return rand.Intn(2-1) + 1 // #nosec G404 - this is just a test
			}

			It("never runs into a deadlock or an unexpected error situation", func(ctx context.Context) {
				// when a deadlock occurs, Ginkgo will abort this test due to a timeout

				for i := 0; i < headbangs; i++ {
					rndDuration := randomBetween(1*time.Millisecond, maxOccupyDuration)
					rndWait := randomBetween(1*time.Millisecond, maxWait)

					switch OccupyOrStop() {
					case Occupy:
						if err := diskOccupier.Occupy(oneHundredKB, rndDuration); err != nil {
							// these are the errors that are allowed to occur
							Expect(err).To(MatchError(errors.New("disk space is already being occupied")))
						}
					case Stop:
						diskOccupier.Stop()
					}

					time.Sleep(rndWait)
				}
			})
		})
	})
})

func isGone(filePath string) bool {
	gone := false
	if _, err := os.Stat(filePath); err != nil && errors.Is(err, os.ErrNotExist) {
		gone = true
	}
	return gone
}

func randomBetween(min time.Duration, max time.Duration) time.Duration {
	return time.Duration(rand.Int63n(int64(max-min)) + int64(min)) // #nosec G404 - this is just a test
}
