package activator

import (
	"encoding/json"
	"fmt"
	"os"
)

// SelfBackend is the Gorouter-reachable, mTLS backend tuple for this activator
// instance, derived from the CF instance environment. It is exactly what the
// cell's own route-emitter advertises for the activator's own route:
//
//	Host                = CF_INSTANCE_IP        (the Diego cell IP, routable by Gorouter)
//	TLSPort             = CF_INSTANCE_PORTS[].external_tls_proxy (host-side TLS NAT port)
//	ServerCertDomainSAN = CF_INSTANCE_GUID      (matches the instance-identity cert SAN)
//
// Publishing these under another app's route (via NATS router.register) makes
// Gorouter route that hostname to this activator instance over mTLS with route
// integrity intact.
type SelfBackend struct {
	Host                string
	TLSPort             int
	ServerCertDomainSAN string
	InstanceGUID        string
}

// cfInstancePort mirrors the entries of the CF_INSTANCE_PORTS JSON array.
type cfInstancePort struct {
	Internal         int `json:"internal"`
	External         int `json:"external"`
	ExternalTLSProxy int `json:"external_tls_proxy"`
	InternalTLSProxy int `json:"internal_tls_proxy"`
}

// SelfBackendFromEnv builds the activator's backend tuple from its CF instance
// environment. It selects the external_tls_proxy port mapped to the app's
// internal listen port (8080), which is the TLS port Gorouter connects to.
func SelfBackendFromEnv() (SelfBackend, error) {
	ip := os.Getenv("CF_INSTANCE_IP")
	guid := os.Getenv("CF_INSTANCE_GUID")
	portsRaw := os.Getenv("CF_INSTANCE_PORTS")
	if ip == "" || guid == "" || portsRaw == "" {
		return SelfBackend{}, fmt.Errorf("CF instance env not available (CF_INSTANCE_IP/GUID/PORTS); activator must run on CF")
	}

	var ports []cfInstancePort
	if err := json.Unmarshal([]byte(portsRaw), &ports); err != nil {
		return SelfBackend{}, fmt.Errorf("failed parsing CF_INSTANCE_PORTS %q: %w", portsRaw, err)
	}

	tlsPort := 0
	for _, p := range ports {
		// The web listener is on internal port 8080; its external_tls_proxy is
		// the host-side NAT port Gorouter dials for TLS.
		if p.Internal == webInternalPort && p.ExternalTLSProxy != 0 {
			tlsPort = p.ExternalTLSProxy
			break
		}
	}
	if tlsPort == 0 {
		// Fall back to the first entry that has an external_tls_proxy.
		for _, p := range ports {
			if p.ExternalTLSProxy != 0 {
				tlsPort = p.ExternalTLSProxy
				break
			}
		}
	}
	if tlsPort == 0 {
		return SelfBackend{}, fmt.Errorf("no external_tls_proxy port found in CF_INSTANCE_PORTS %q", portsRaw)
	}

	return SelfBackend{
		Host:                ip,
		TLSPort:             tlsPort,
		ServerCertDomainSAN: guid,
		InstanceGUID:        guid,
	}, nil
}

const webInternalPort = 8080
