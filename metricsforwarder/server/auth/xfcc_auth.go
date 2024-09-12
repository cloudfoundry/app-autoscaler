package auth

import (
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/db"
	"code.cloudfoundry.org/lager/v3"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var ErrXFCCHeaderNotFound = errors.New("mTLS authentication method not found")
var ErrorNoAppIDFound = errors.New("certificate does not contain an app id")
var ErrorAppIDWrong = errors.New("app is not allowed to send metrics due to invalid app id in certificate")

var ErrorAppNotBound = errors.New("application is not bound to the same service instance")

func (a *Auth) XFCCAuth(r *http.Request, bindingDB db.BindingDB, appID string) error {
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

	submitterAppCert := readAppIdFromCert(cert)

	if len(submitterAppCert) == 0 {
		return ErrorNoAppIDFound
	}

	// appID = custom metrics producer
	// submitterAppCert = app id in certificate
	// Case 1 : custom metrics can only be published by the app itself
	// Case 2 : custom metrics can be published by any app bound to the same autoscaler instance
	// In short, if the requester is not same as the scaling app
	if appID != submitterAppCert {

		// check for case 2 here
		/*
			TODO
			Read the parameter new boolean parameter from the http request body named as "allow_from": "bound_app or same_app"
			If it is set to true, then
			- check if the app is bound to the same autoscaler instance - How to check this? check from the database binding_db table -> app_id->binding_id->service_instance_id-> all bound apps
			- if it is bound, then allow the request i.e custom metrics to be published
			- if it is not bound, then return an error saying "app is not allowed to send custom metrics on as it not bound to the autoscaler service instance"
			If the parameter is not set, then follow the existing logic and allow the request to be published


		*/
		a.logger.Info("Checking custom metrics submission strategy")
		validSubmitter, err := verifyMetricSubmissionStrategy(r, a.logger, bindingDB, submitterAppCert, appID)
		if err != nil {
			a.logger.Error("error-verifying-custom-metrics-submitter-app", err, lager.Data{"metric-submitter-app-id": submitterAppCert})
			return err
		} /*  no need to check as this is the default case
		else if customMetricSubmissionStrategy == "same_app" || customMetricSubmissionStrategy == "" { // default case
			// if the app is the same app, then allow the request to the next handler i.e 403
			a.logger.Info("custom-metrics-submission-strategy", lager.Data{"strategy": customMetricSubmissionStrategy})
			return ErrorAppIDWrong
		} */
		if validSubmitter == true {
			return nil
		} else {
			return ErrorAppIDWrong
		}
	}
	return nil
}

func verifyMetricSubmissionStrategy(r *http.Request, logger lager.Logger, bindingDB db.BindingDB, submitterAppCert string, appID string) (bool, error) {

	customMetricSubmissionStrategy := r.Header.Get("custom-metrics-submission-strategy")
	customMetricSubmissionStrategy = strings.ToLower(customMetricSubmissionStrategy)
	if customMetricSubmissionStrategy == "" {
		logger.Info("custom-metrics-submission-strategy-not-found", lager.Data{"appID": appID})
		return false, nil
	}
	if customMetricSubmissionStrategy == "bound_app" {
		logger.Info("custom-metrics-submission-strategy-found", lager.Data{"appID": appID, "strategy": customMetricSubmissionStrategy})
		// check if the app is bound to same autoscaler instance by check the binding id from the bindingdb
		// if the app is bound to the same autoscaler instance, then allow the request to the next handler i.e publish custom metrics
		isAppBound, err := bindingDB.IsAppBoundToSameAutoscaler(r.Context(), submitterAppCert, appID)
		if err != nil {
			logger.Error("error-checking-app-bound-to-same-service", err, lager.Data{"metric-submitter-app-id": submitterAppCert})
			return false, err
		}
		if isAppBound == false {
			logger.Info("app-not-bound-to-same-service", lager.Data{"app-id": submitterAppCert})
			return false, ErrorAppNotBound
		}
	}
	return true, nil
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
