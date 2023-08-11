package cf_test

import (
	"encoding/json"
	"errors"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/cf"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Errors test", func() {
	url := "https://some.cf.install/v3/apps/guid/processes/web/action/scale"
	Context("nil type error", func() {
		It("should be invalid", func() {
			var err *cf.CfError
			Expect(err.IsValid()).To(BeFalse())
		})
		It("should IsNotFound == false ", func() {
			var err *cf.CfError
			Expect(err.IsNotFound()).To(BeFalse())
		})
	})
	Context("Constructing Errors", func() {
		It("should be able to construct from call", func() {
			err := cf.NewCfError(url, "some-id", 404, []byte(`{"errors": [{"code": 1,"title": "Title","detail": "Detail"}]}`))
			Expect(err).To(BeAssignableToTypeOf(&cf.CfError{}))
			Expect(err.Error()).To(Equal("cf api Error url='https://some.cf.install/v3/apps/guid/processes/web/action/scale', resourceId='some-id': ['Title' code: 1, Detail: 'Detail']"))
		})
		It("should return error if not marshalable", func() {
			err := cf.NewCfError(url, "some-id", 404, []byte(`{"errors": [{"code" "Title","detail": "Detail"}]}`))
			Expect(err).ToNot(BeAssignableToTypeOf(&cf.CfError{}))
			var errType *json.SyntaxError
			Expect(errors.As(err, &errType)).Should(BeTrue(), "Error was: %#v", interface{}(err))
			Expect(err.Error()).To(MatchRegexp("failed to unmarshal id:some-id error status '404' body:'{\\\"errors\\\": "))
		})
		It("should return error if not incorrect Json", func() {
			err := cf.NewCfError(url, "some-id", 404, []byte(`{"Some":"JSON"}`))
			Expect(err).ToNot(BeAssignableToTypeOf(&cf.CfError{}))
			Expect(errors.Is(err, cf.ErrInvalidJson)).To(BeTrue())
			Expect(err.Error()).To(MatchRegexp("invalid cfError: resource some-id status:404 body:{\"Some\":\"JSON\"}"))
		})
	})
	Context("Parsing tests", func() {
		var errorResponse string
		var cfError *cf.CfError
		JustBeforeEach(func() {
			cfError = &cf.CfError{}
			err := json.Unmarshal([]byte(errorResponse), cfError)
			Expect(err).NotTo(HaveOccurred())
		})
		Context("resource not found response", func() {
			BeforeEach(func() {
				errorResponse = `{"errors": [{"detail": "App usage event not found","title": "CF-ResourceNotFound","code": 10010}]}`
			})
			It("Should return true for IsNotFound()", func() {
				Expect(cfError.IsNotFound()).To(BeTrue())
			})

		})
		Context("resource not authorized", func() {
			BeforeEach(func() {
				errorResponse = `{"errors": [{"detail": "The authenticated user does not have permission to perform this operation","title": "CF-NotAuthorized","code": 10003}]}`
			})
			It("Should return true for IsNotAuthorised()", func() {
				Expect(cfError.IsNotAuthorised()).To(BeTrue())
			})
		})
		Context("resource not authenticated", func() {
			BeforeEach(func() {
				errorResponse = `{"errors": [{"detail": "No auth token was given, but authentication is required for this endpoint","title": "CF-NotAuthenticated","code": 10002}]}`
			})
			It("Should return true for IsNotAuthenticated()", func() {
				Expect(cfError.IsNotAuthenticated()).To(BeTrue())
			})
		})
		Context("There is one error", func() {
			BeforeEach(func() { errorResponse = `{"errors": [{"code": 1,"title": "Title","detail": "Detail"}]}` })
			It("Should have the right message", func() {
				Expect(cfError.Error()).To(Equal("cf api Error url='', resourceId='': ['Title' code: 1, Detail: 'Detail']"))
			})
			It("Should be valid", func() {
				Expect(cfError.IsValid()).To(BeTrue())
			})
			It("Should return false for IsNotFound()", func() {
				Expect(cfError.IsNotFound()).To(BeFalse())
			})
		})
		Context("There is two errors", func() {
			BeforeEach(func() {
				errorResponse = `{"errors": [{"code": 1,"title": "Title1","detail": "Detail1"},{"code": 2,"title": "Title2","detail": "Detail2"}]}`
			})
			It("Should have the right message", func() {
				Expect(cfError.Error()).To(Equal("cf api Error url='', resourceId='': ['Title1' code: 1, Detail: 'Detail1'], ['Title2' code: 2, Detail: 'Detail2']"))
			})
			It("Should be valid", func() {
				Expect(cfError.IsValid()).To(BeTrue())
			})
			It("Should return false for IsNotFound()", func() {
				Expect(cfError.IsNotFound()).To(BeFalse())
			})
		})
		Context("There is two errors with one Notfound", func() {
			BeforeEach(func() {
				errorResponse = `{"errors": [{"code": 1,"title": "Title1","detail": "Detail1"},{"detail": "App usage event not found","title": "CF-ResourceNotFound","code": 10010}]}`
			})
			It("Should return true for IsNotFound()", func() {
				Expect(cfError.IsNotFound()).To(BeTrue())
			})
		})
		Context("There is no errors", func() {
			BeforeEach(func() {
				errorResponse = `{"errors": []}`
			})
			It("Should have the right message", func() {
				Expect(cfError.Error()).To(Equal("cf api Error url='', resourceId='': None found"))
			})
			It("Should be invalid", func() {
				Expect(cfError.IsValid()).To(BeFalse())
			})
			It("Should return false for IsNotFound()", func() {
				Expect(cfError.IsNotFound()).To(BeFalse())
			})
		})
	})
})
