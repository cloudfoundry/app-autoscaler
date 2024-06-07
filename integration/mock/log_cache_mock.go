package mock

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	rpc "code.cloudfoundry.org/go-log-cache/v2/rpc/logcache_v1"
	"code.cloudfoundry.org/go-loggregator/v9/rpc/loggregator_v2"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type LogCache struct {
	mutex sync.Mutex

	lis net.Listener

	readResponses map[string]*rpc.ReadResponse
	readErrors    map[string]error

	instantQueryResults map[string]*rpc.PromQL_InstantQueryResult
	instantQueryErrors  map[string]error

	tlsConfig *tls.Config

	rpc.UnimplementedEgressServer
	rpc.UnimplementedIngressServer
	rpc.UnimplementedPromQLQuerierServer
}

func NewLogCache(tlsConfig *tls.Config) *LogCache {
	return &LogCache{
		tlsConfig: tlsConfig,

		readResponses: map[string]*rpc.ReadResponse{},
		readErrors:    map[string]error{},

		instantQueryResults: map[string]*rpc.PromQL_InstantQueryResult{},
		instantQueryErrors:  map[string]error{},
	}
}

func (s *LogCache) Start(port int) string {
	var err error
	s.lis, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var srv *grpc.Server
	if s.tlsConfig == nil {
		srv = grpc.NewServer()
	} else {
		srv = grpc.NewServer(grpc.Creds(credentials.NewTLS(s.tlsConfig)))
	}
	rpc.RegisterIngressServer(srv, s)
	rpc.RegisterEgressServer(srv, s)
	rpc.RegisterPromQLQuerierServer(srv, s)

	//nolint:errcheck
	go srv.Serve(s.lis)

	return s.lis.Addr().String()
}

func (s *LogCache) Stop() {
	s.lis.Close()
}

func (s *LogCache) Read(ctx context.Context, r *rpc.ReadRequest) (*rpc.ReadResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	response := &rpc.ReadResponse{Envelopes: &loggregator_v2.EnvelopeBatch{Batch: []*loggregator_v2.Envelope{}}}
	if storedResponse, ok := s.readResponses[r.SourceId]; ok {
		response = storedResponse
	}

	err := errors.New("default-error")
	if storedError, ok := s.readErrors[r.SourceId]; ok {
		err = storedError
	}

	return response, err
}

func (s *LogCache) ReadReturns(sourceId string, response *rpc.ReadResponse, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.readResponses[sourceId] = response
	s.readErrors[sourceId] = err
}

func (s *LogCache) InstantQuery(ctx context.Context, r *rpc.PromQL_InstantQueryRequest) (*rpc.PromQL_InstantQueryResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// every query carries the source_id, example "http{source_id='foo'}"
	// find stored mock-result or mock-error by checking if any of the stored source ids is contained in the query
	query := r.Query
	for sourceId := range s.instantQueryResults {
		if strings.Contains(query, sourceId) {
			return s.instantQueryResults[sourceId], s.instantQueryErrors[sourceId]
		}
	}

	return nil, fmt.Errorf("default-error(query: %s)", query)
}

func (s *LogCache) InstantQueryReturns(sourceId string, result *rpc.PromQL_InstantQueryResult, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.instantQueryResults[sourceId] = result
	s.instantQueryErrors[sourceId] = err
}

func NewTLSConfig(caPath, certPath, keyPath, cn string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         cn,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: false,
	}

	caCertBytes, err := os.ReadFile(caPath)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCertBytes); !ok {
		return nil, errors.New("cannot parse ca cert")
	}

	tlsConfig.RootCAs = caCertPool

	return tlsConfig, nil
}
