package activator

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/models"

	"github.com/nats-io/nats.go"

	"code.cloudfoundry.org/lager/v3"
)

const (
	natsRegisterSubject   = "router.register"
	natsUnregisterSubject = "router.unregister"
	// defaultStaleThreshold is what Gorouter uses to prune an endpoint that has
	// not been refreshed; we advertise it and refresh well within it.
	defaultStaleThresholdSeconds = 120
	defaultRefreshInterval       = 20 * time.Second
)

// Registrar registers/unregisters the activator as a backend for parked-app
// route URIs on the Gorouter NATS bus, keeping the registration fresh so the
// route stays alive while the app has zero instances.
type Registrar interface {
	Register(uris []string) error
	Unregister(uris []string) error
	// Run keeps all currently-registered URIs fresh until the context is done.
	Run(stop <-chan struct{})
}

// registerMessage is the router.register / router.unregister NATS payload
// (subset of gorouter's RegistryMessage). tls_port + server_cert_domain_san
// make Gorouter connect to the backend over mTLS with route-integrity.
type registerMessage struct {
	URIs                    []string `json:"uris"`
	Host                    string   `json:"host"`
	TLSPort                 int      `json:"tls_port,omitempty"`
	App                     string   `json:"app,omitempty"`
	PrivateInstanceID       string   `json:"private_instance_id,omitempty"`
	ServerCertDomainSAN     string   `json:"server_cert_domain_san,omitempty"`
	StaleThresholdInSeconds int      `json:"stale_threshold_in_seconds,omitempty"`
}

type natsRegistrar struct {
	logger          lager.Logger
	conn            *nats.Conn
	self            SelfBackend
	refreshInterval time.Duration
	staleThreshold  int

	mu   sync.Mutex
	uris map[string]struct{} // set of currently-registered route URIs
}

// NatsConfig configures the NATS connection used for route registration.
type NatsConfig struct {
	URLs           []string        `yaml:"urls" json:"urls"`
	TLS            models.TLSCerts `yaml:"tls" json:"tls"`
	Username       string          `yaml:"username" json:"username"`
	Password       string          `yaml:"password" json:"password"`
	RefreshSeconds int             `yaml:"refresh_seconds" json:"refresh_seconds"`
	StaleSeconds   int             `yaml:"stale_seconds" json:"stale_seconds"`
}

// NewNatsRegistrar dials NATS (mTLS) and returns a Registrar that advertises the
// given self backend tuple for parked routes.
func NewNatsRegistrar(logger lager.Logger, conf NatsConfig, self SelfBackend) (Registrar, error) {
	log := logger.Session("nats-registrar")
	opts := []nats.Option{
		nats.Name("autoscaler-activator"),
		nats.MaxReconnects(-1),
		// Do not crash the activator if NATS is briefly unavailable: keep
		// retrying the initial connect and all reconnects in the background.
		nats.RetryOnFailedConnect(true),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Error("nats-disconnected", err)
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			log.Info("nats-reconnected", lager.Data{"url": c.ConnectedUrl()})
		}),
	}
	if conf.Username != "" {
		opts = append(opts, nats.UserInfo(conf.Username, conf.Password))
	}
	if conf.TLS.CertFile != "" || conf.TLS.CACertFile != "" {
		tlsCfg, err := conf.TLS.CreateClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed building NATS TLS config: %w", err)
		}
		opts = append(opts, nats.Secure(tlsCfg))
	} else {
		opts = append(opts, nats.Secure(&tls.Config{MinVersion: tls.VersionTLS12})) //nolint:gosec // MinVersion set
	}

	// With RetryOnFailedConnect, Connect returns a usable (reconnecting) conn
	// even if the server is down right now, so a NATS outage never fails
	// activator startup — only genuine option errors do.
	conn, err := nats.Connect(joinURLs(conf.URLs), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed initialising NATS connection: %w", err)
	}

	refresh := defaultRefreshInterval
	if conf.RefreshSeconds > 0 {
		refresh = time.Duration(conf.RefreshSeconds) * time.Second
	}
	stale := defaultStaleThresholdSeconds
	if conf.StaleSeconds > 0 {
		stale = conf.StaleSeconds
	}

	return &natsRegistrar{
		logger:          log,
		conn:            conn,
		self:            self,
		refreshInterval: refresh,
		staleThreshold:  stale,
		uris:            make(map[string]struct{}),
	}, nil
}

func (r *natsRegistrar) Register(uris []string) error {
	r.mu.Lock()
	for _, u := range uris {
		r.uris[u] = struct{}{}
	}
	r.mu.Unlock()
	return r.publish(natsRegisterSubject, uris)
}

func (r *natsRegistrar) Unregister(uris []string) error {
	r.mu.Lock()
	for _, u := range uris {
		delete(r.uris, u)
	}
	r.mu.Unlock()
	return r.publish(natsUnregisterSubject, uris)
}

func (r *natsRegistrar) Run(stop <-chan struct{}) {
	ticker := time.NewTicker(r.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			r.conn.Close()
			return
		case <-ticker.C:
			r.mu.Lock()
			uris := make([]string, 0, len(r.uris))
			for u := range r.uris {
				uris = append(uris, u)
			}
			r.mu.Unlock()
			if len(uris) == 0 {
				continue
			}
			if err := r.publish(natsRegisterSubject, uris); err != nil {
				r.logger.Error("failed-refresh-register", err, lager.Data{"uris": uris})
			}
		}
	}
}

func (r *natsRegistrar) publish(subject string, uris []string) error {
	if len(uris) == 0 {
		return nil
	}
	msg := registerMessage{
		URIs:                    uris,
		Host:                    r.self.Host,
		TLSPort:                 r.self.TLSPort,
		PrivateInstanceID:       r.self.InstanceGUID,
		ServerCertDomainSAN:     r.self.ServerCertDomainSAN,
		StaleThresholdInSeconds: r.staleThreshold,
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed marshalling %s message: %w", subject, err)
	}
	if err := r.conn.Publish(subject, payload); err != nil {
		return fmt.Errorf("failed publishing %s: %w", subject, err)
	}
	return r.conn.Flush()
}

func joinURLs(urls []string) string {
	return strings.Join(urls, ",")
}
