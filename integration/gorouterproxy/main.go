package main

import (
	"crypto/tls"
	"encoding/pem"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"flag"
	"fmt"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers/auth"
)

var (
	port      = flag.String("port", "8888", "Port for xfcc proxy")
	forwardTo = flag.String("forwardTo", "", "Port to forward to")
	keyFile   = flag.String("keyFile", "", "Path to key file")
	certFile  = flag.String("certFile", "", "Path to cert file")
	logger    = log.New(os.Stdout, "gorouter-proxy", log.LstdFlags)
)

func main() {
	flag.Parse()
	startServer()
}

func startServer() {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", *port),
		Handler: http.HandlerFunc(forwardHandler),
		TLSConfig: &tls.Config{
			ClientAuth: tls.RequireAnyClientCert,
			MinVersion: tls.VersionTLS12,
		},
		ReadHeaderTimeout: 10 * time.Second,
	}

	if !fileExists(*certFile) || !fileExists(*keyFile) {
		logger.Printf("Cert or key file does not exist: cert=%s, key=%s", *certFile, *keyFile)
		return
	}

	logger.Printf("gorouter-proxy.started - port %s, forwardTo %s", *port, *forwardTo)
	if err := server.ListenAndServeTLS(*certFile, *keyFile); err != nil {
		logger.Printf("Error starting server: %v", err)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

func forwardHandler(w http.ResponseWriter, inRequest *http.Request) {
	var body []byte
	var err error

	tls := inRequest.TLS
	if !isClientCertValid(tls) {
		http.Error(w, "No client certificate", http.StatusForbidden)
		return
	}

	cert := createCert(tls)
	if cert == nil {
		http.Error(w, "Failed to parse client certificate", http.StatusInternalServerError)
		return
	}

	resp, err := forwardRequest(inRequest, cert)
	if err != nil {
		logger.Printf("Error forwarding request: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	if body, err = io.ReadAll(resp.Body); err != nil {
		logger.Printf("Error reading response: %v", err)
		return
	}

	if _, err := w.Write(body); err != nil {
		logger.Printf("Error writing response: %v", err)
	}
}

func isClientCertValid(tls *tls.ConnectionState) bool {
	if tls == nil || len(tls.PeerCertificates) == 0 {
		logger.Printf("No client certificate")
		return false
	}
	return true
}

func createCert(tls *tls.ConnectionState) *auth.Cert {
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: tls.PeerCertificates[0].Raw,
	})
	return auth.NewCert(string(pemData))
}

func forwardRequest(inRequest *http.Request, cert *auth.Cert) (*http.Response, error) {
	client := &http.Client{}
	logger.Printf("Forwarding request to %s", *forwardTo)

	// Always forward to HTTP as any component on the other side of the router running
	// on a CF container will be using HTTP
	url := inRequest.URL
	url.Scheme = "http"
	url.Host = fmt.Sprintf("127.0.0.1:%s", *forwardTo)

	outRequest, err := http.NewRequest(inRequest.Method, url.String(), inRequest.Body)
	outRequest.Header = inRequest.Header
	if err != nil {
		logger.Printf("Error creating request: %v", err)
		return nil, err
	}

	if cert.GetXFCCHeader() == "" {
		log.Printf("XFCC header is empty")
		return nil, fmt.Errorf("XFCC header is empty")
	} else {
		log.Printf("Adding XFCC header before forwarding request")

		outRequest.Header.Add("X-Forwarded-Client-Cert", cert.GetXFCCHeader())
		resp, err := client.Do(outRequest)
		return resp, err
	}
}
