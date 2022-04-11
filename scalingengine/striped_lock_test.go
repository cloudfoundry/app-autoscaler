package scalingengine_test

import (
	. "code.cloudfoundry.org/app-autoscaler/src/autoscaler/scalingengine"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StripedLock", func() {
	var (
		stripedLock *StripedLock
		capacity    int
		err         interface{}
	)
	Describe("NewStripedLock", func() {
		JustBeforeEach(func() {
			defer func() {
				err = recover()
			}()
			stripedLock = NewStripedLock(capacity)
		})
		Context("when creating a striped lock with invalid capacity", func() {
			BeforeEach(func() {
				capacity = -1
			})
			It("panics", func() {
				Expect(err).To(Equal("invalid striped lock capacity"))
			})
		})

		Context("when creating a striped lock with valid capacity", func() {
			BeforeEach(func() {
				capacity = 32
			})
			It("returns the striped lock", func() {
				Expect(stripedLock).NotTo(BeNil())
				Expect(err).To(BeNil())
			})
		})

	})

	Describe("GetLock", func() {
		BeforeEach(func() {
			stripedLock = NewStripedLock(4)
		})

		Context("when getting lock with the same key", func() {
			It("returns the same lock", func() {
				Expect(stripedLock.GetLock("some-key")).To(BeIdenticalTo(stripedLock.GetLock("some-key")))
			})
		})
	})
})
