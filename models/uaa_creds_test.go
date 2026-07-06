package models_test

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("UAACreds.IsPasswordGrant",
	func(grantType string, expected bool) {
		creds := models.UAACreds{GrantType: grantType}
		Expect(creds.IsPasswordGrant()).To(Equal(expected))
	},
	Entry("returns true for password grant", models.GrantTypePassword, true),
	Entry("returns false for empty grant type", "", false),
	Entry("returns false for client_credentials", "client_credentials", false),
)
