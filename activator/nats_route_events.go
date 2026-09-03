package activator

import (
	"encoding/json"

	"github.com/nats-io/nats.go"

	"code.cloudfoundry.org/lager/v3"
)

// natsRouteEventSubscriber is a RouteEventSubscriber backed by the Gorouter NATS
// bus. It subscribes to router.register (the same subject route-emitter uses to
// advertise an app's backends) and, for every registration of a parked route by
// a backend that is NOT this activator, emits an Upsert RouteEvent — the "the
// real app is up" readiness signal.
//
// This reuses the registrar's existing NATS connection: readiness and route
// keep-alive share one bus and one mTLS connection, so there is no separate
// routing-api HTTP stream (and no UAA dependency) to authenticate. See
// docs/design/scale-to-zero.md §5.3.
type natsRouteEventSubscriber struct {
	logger lager.Logger
	conn   *nats.Conn
	self   SelfBackend
}

// NewNatsRouteEventSubscriber builds a readiness subscriber over conn, filtering
// out the activator's own backend registrations using self.
func NewNatsRouteEventSubscriber(logger lager.Logger, conn *nats.Conn, self SelfBackend) RouteEventSubscriber {
	return &natsRouteEventSubscriber{
		logger: logger.Session("nats-route-events"),
		conn:   conn,
		self:   self,
	}
}

func (s *natsRouteEventSubscriber) Subscribe() (RouteEventStream, error) {
	stream := &natsRouteEventStream{
		events: make(chan RouteEvent, 64),
		errs:   make(chan error, 1),
	}
	sub, err := s.conn.Subscribe(natsRegisterSubject, func(msg *nats.Msg) {
		s.handle(msg, stream)
	})
	if err != nil {
		return nil, err
	}
	stream.sub = sub
	return stream, nil
}

// handle decodes a router.register message and, if it registers a parked route
// from a non-self backend, enqueues one Upsert per URI.
func (s *natsRouteEventSubscriber) handle(msg *nats.Msg, stream *natsRouteEventStream) {
	events, err := decodeReadinessEvents(msg.Data, s.self)
	if err != nil {
		s.logger.Error("failed-to-decode-register-message", err)
		return
	}
	for _, e := range events {
		select {
		case stream.events <- e:
		default:
			s.logger.Info("readiness-event-buffer-full-dropping", lager.Data{"uri": e.RouteURL})
		}
	}
}

// decodeReadinessEvents turns a router.register payload into the readiness
// events it implies: one Upsert per URI, unless the registration is the
// activator's own backend (self), in which case none. Returns an error only if
// the payload is not valid JSON.
func decodeReadinessEvents(data []byte, self SelfBackend) ([]RouteEvent, error) {
	var rm registerMessage
	if err := json.Unmarshal(data, &rm); err != nil {
		return nil, err
	}
	if isSelfBackend(rm, self) {
		return nil, nil // our own keep-alive registration is not a readiness signal
	}
	events := make([]RouteEvent, 0, len(rm.URIs))
	for _, uri := range rm.URIs {
		events = append(events, RouteEvent{RouteURL: uri, Action: actionUpsert})
	}
	return events, nil
}

// isSelfBackend reports whether a registration advertises this activator
// instance as the backend. The instance GUID (private_instance_id /
// server_cert_domain_san) is the authoritative identity; host+tls_port is a
// fallback for messages that omit it.
func isSelfBackend(rm registerMessage, self SelfBackend) bool {
	if self.InstanceGUID != "" &&
		(rm.PrivateInstanceID == self.InstanceGUID || rm.ServerCertDomainSAN == self.ServerCertDomainSAN) {
		return true
	}
	return rm.Host == self.Host && rm.TLSPort == self.TLSPort
}

// natsRouteEventStream adapts a NATS subscription to RouteEventStream.
type natsRouteEventStream struct {
	sub    *nats.Subscription
	events chan RouteEvent
	errs   chan error
}

func (s *natsRouteEventStream) Next() (RouteEvent, error) {
	select {
	case e := <-s.events:
		return e, nil
	case err := <-s.errs:
		return RouteEvent{}, err
	}
}

func (s *natsRouteEventStream) Close() error {
	if s.sub != nil {
		return s.sub.Unsubscribe()
	}
	return nil
}
