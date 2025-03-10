package auth

import (
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
)

var ErrXFCCHeaderNotFound = errors.New("mTLS authentication method not found")
var ErrorNoAppIDFound = errors.New("certificate does not contain an app id")
var ErrorAppIDWrong = errors.New("app is not allowed to send metrics due to invalid app id in certificate")
var ErrorAppNotBound = errors.New("application is not bound to the same service instance")

// TODO: find common functionality in src/autoscaler/helpers/auth/xfcc_auth.go and reuse it
func (a *Auth) XFCCAuth(r *http.Request, bindingDB db.BindingDB, appToScaleID string) error {
	xfccHeader := r.Header.Get("X-Forwarded-Client-Cert")
	if xfccHeader == "" {
		return ErrXFCCHeaderNotFound
	}

	data, err := base64.StdEncoding.DecodeString(removeQuotes(xfccHeader))
	if err != nil {
		return fmt.Errorf("base64 parsing failed: %w", err)
	}

	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	submitterAppIDFromCert := readAppIdFromCert(cert)

	if len(submitterAppIDFromCert) == 0 {
		return ErrorNoAppIDFound
	}

	// Case: Submitting app is not the same as the app to scale
	if appToScaleID != submitterAppIDFromCert {
		var metricSubmissionStrategy MetricsSubmissionStrategy
		customMetricSubmissionStrategy, err := bindingDB.GetCustomMetricStrategyByAppId(r.Context(), appToScaleID)
		if err != nil {
			a.logger.Error("failed-to-get-custom-metric-strategy", err, lager.Data{"appToScaleID": appToScaleID})
			return err
		}
		a.logger.Info("custom-metrics-submission-strategy", lager.Data{"appToScaleID": appToScaleID, "submitterAppIDFromCert": submitterAppIDFromCert, "strategy": customMetricSubmissionStrategy})

		if customMetricSubmissionStrategy == models.CustomMetricsBoundApp {
			metricSubmissionStrategy = &BoundedMetricsSubmissionStrategy{}
		} else {
			metricSubmissionStrategy = &DefaultMetricsSubmissionStrategy{}
		}
		err = metricSubmissionStrategy.validate(appToScaleID, submitterAppIDFromCert, a.logger, bindingDB, r)
		if err != nil {
			return err
		}
	}

	return nil
}

func readAppIdFromCert(cert *x509.Certificate) string {
	var certAppId string
	for _, ou := range cert.Subject.OrganizationalUnit {
		if strings.Contains(ou, "app:") {
			certAppId = strings.Split(ou, ":")[1]
			break
		}
	}
	return certAppId
}

func removeQuotes(xfccHeader string) string {
	if xfccHeader[0] == '"' {
		xfccHeader = xfccHeader[1 : len(xfccHeader)-1]
	}
	return xfccHeader
}
