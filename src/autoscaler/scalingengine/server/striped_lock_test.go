package server_test

import (
	. "autoscaler/scalingengine/server"

	. "github.com/onsi/ginkgo"
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
			stripedLock = NewStripedLock(32)
		})

		Context("when getting lock the first time", func() {
			It("creates the lock and returns", func() {
				Expect(stripedLock.GetLock("some key")).NotTo(BeNil())
			})
		})

		Context("when getting lock with the same key", func() {
			It("returns the same lock", func() {
				Expect(stripedLock.GetLock("some-key")).Should(BeIdenticalTo(stripedLock.GetLock("some-key")))
			})
		})

	})
})
