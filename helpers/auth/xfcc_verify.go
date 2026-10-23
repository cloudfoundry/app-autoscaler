package auth

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"
	"code.cloudfoundry.org/lager/v3"
)

var ErrChainVerificationFailed = errors.New("XFCC certificate chain verification failed")

const trustRefreshInterval = 30 * time.Second

// ChainVerifier loads immutable container roots once and refreshes the mounted
// instance chain at most once per interval. It never reads private keys or uses
// OS default roots. Missing trust fails closed, except in explicit shadow mode.
type ChainVerifier struct {
	mode, instancePath       string
	logger                   lager.Logger
	roots                    *x509.CertPool
	rootErr                  error
	mu                       sync.Mutex
	intermediates            *x509.CertPool
	refreshErr               error
	nextRefresh, nextWarning time.Time
	now                      func() time.Time
}

func NewChainVerifier(logger lager.Logger, config models.XFCCAuth) (*ChainVerifier, error) {
	mode := config.VerificationMode
	if mode == "" {
		mode = "enforce"
	}
	if mode != "enforce" && mode != "shadow" && mode != "off" {
		return nil, fmt.Errorf("invalid XFCC verification mode %q", mode)
	}
	v := &ChainVerifier{mode: mode, logger: logger, now: time.Now}
	if mode != "off" {
		v.instancePath = trustPath(config.InstanceCertPath, "CF_INSTANCE_CERT", "/etc/cf-instance-credentials/instance.crt")
		v.roots, v.rootErr = loadRoots(trustPath(config.SystemCertPath, "CF_SYSTEM_CERT_PATH", "/etc/cf-system-certificates"))
	}
	return v, nil
}

func trustPath(configured, environment, fallback string) string {
	if configured != "" {
		return configured
	}
	if path := os.Getenv(environment); path != "" {
		return path
	}
	return fallback
}

func loadRoots(dir string) (*x509.CertPool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		certs, err := readCertificates(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, err
		}
		for _, cert := range certs {
			if cert.IsCA {
				pool.AddCert(cert)
				count++
			}
		}
	}
	if count == 0 {
		return nil, errors.New("no XFCC trust anchors")
	}
	return pool, nil
}

func readCertificates(path string) ([]*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var certs []*x509.Certificate
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			return certs, nil
		}
		data = rest
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
}

func (v *ChainVerifier) Verify(leaf *x509.Certificate) error {
	if v.mode == "off" {
		return nil
	}
	now := v.now()
	v.mu.Lock()
	if !now.Before(v.nextRefresh) {
		v.nextRefresh = now.Add(trustRefreshInterval)
		v.intermediates, v.refreshErr = v.loadIntermediates()
	}
	pool, err := v.intermediates, v.rootErr
	if err == nil {
		err = v.refreshErr
	}
	v.mu.Unlock()
	if err == nil {
		if leaf.IsCA {
			err = errors.New("XFCC caller certificate is a CA")
		} else {
			_, err = leaf.Verify(x509.VerifyOptions{
				Roots:         v.roots,
				Intermediates: pool,
				CurrentTime:   now,
				KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			})
		}
	}
	if err == nil {
		v.logger.Debug("xfcc-chain-verified")
		return nil
	}
	if v.mode == "shadow" {
		v.mu.Lock()
		if !now.Before(v.nextWarning) {
			v.nextWarning = now.Add(trustRefreshInterval)
			// Do not log attacker-controlled certificate contents or subjects.
			v.logger.Info("xfcc-chain-verification-warning", lager.Data{"mode": "shadow", "result": "failed"})
		}
		v.mu.Unlock()
		return nil
	}
	return fmt.Errorf("%w: %w", ErrChainVerificationFailed, err)
}

func (v *ChainVerifier) loadIntermediates() (*x509.CertPool, error) {
	certs, err := readCertificates(v.instancePath)
	if err != nil {
		return nil, err
	}
	if len(certs) < 2 {
		return nil, errors.New("instance certificate chain has no intermediate")
	}
	pool := x509.NewCertPool()
	for _, cert := range certs[1:] {
		if !cert.IsCA {
			return nil, errors.New("instance certificate chain contains a non-CA issuer")
		}
		pool.AddCert(cert)
	}
	return pool, nil
}
