package collector

import (
	"net/http"
	"sync"
	"time"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"

	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"code.cloudfoundry.org/lager"
	"github.com/golang/protobuf/proto" //nolint
	"github.com/gorilla/websocket"
)

type wsMessageHandler struct {
	logger           lager.Logger
	envelopeChannels []chan *loggregator_v2.Envelope
	keepAlive        time.Duration
	lock             *sync.Mutex
}

func NewWSMessageHandler(logger lager.Logger, envelopeChannels []chan *loggregator_v2.Envelope, keepAlive time.Duration) *wsMessageHandler {
	return &wsMessageHandler{
		logger:           logger,
		envelopeChannels: envelopeChannels,
		keepAlive:        keepAlive,
		lock:             &sync.Mutex{},
	}
}

func (h *wsMessageHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	ws, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		h.logger.Error("serve-websocket-upgrade", err)
		return
	}
	defer ws.Close()

	closeCode, closeMessage := h.runWebsocketUntilClosed(ws)
	err = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, closeMessage), time.Time{})
	if err != nil {
		h.logger.Error("serve-websocket-close", err)
		return
	}
}

func (h *wsMessageHandler) runWebsocketUntilClosed(ws *websocket.Conn) (closeCode int, closeMessage string) {
	keepAliveExpired := make(chan struct{})
	clientWentAway := make(chan struct{})

	go func() {
		for {
			_, bytes, err := ws.ReadMessage()
			if err != nil {
				h.logger.Error("run-websocket-read-message", err)
				close(clientWentAway)
				return
			}
			var envelop loggregator_v2.Envelope
			err = proto.Unmarshal(bytes, &envelop)
			if err != nil {
				h.logger.Error("run-websocket-unmarshal", err)
			}
			h.envelopeChannels[helpers.FNVHash(envelop.GetSourceId())%uint32(len(h.envelopeChannels))] <- &envelop
		}
	}()

	go func() {
		NewKeepAlive(h.lock, ws, h.keepAlive).Run()
		close(keepAliveExpired)
	}()

	closeCode = websocket.CloseNormalClosure
	closeMessage = ""
	for {
		select {
		case <-clientWentAway:
			return
		case <-keepAliveExpired:
			closeCode = websocket.ClosePolicyViolation
			closeMessage = "Client did not respond to ping before keep-alive timeout expired."
			return
		}
	}
}

type KeepAlive struct {
	lock              *sync.Mutex
	conn              *websocket.Conn
	pongChan          chan struct{}
	keepAliveInterval time.Duration
}

func NewKeepAlive(lock *sync.Mutex, conn *websocket.Conn, keepAliveInterval time.Duration) *KeepAlive {
	return &KeepAlive{
		lock:              lock,
		conn:              conn,
		pongChan:          make(chan struct{}, 1),
		keepAliveInterval: keepAliveInterval,
	}
}

func (k *KeepAlive) Run() {
	k.lock.Lock()
	k.conn.SetPongHandler(k.pongHandler)
	k.lock.Unlock()

	defer func() {
		k.lock.Lock()
		k.conn.SetPongHandler(nil)
		k.lock.Unlock()
	}()

	timeout := time.NewTimer(k.keepAliveInterval)
	for {
		err := k.conn.WriteControl(websocket.PingMessage, nil, time.Time{})
		if err != nil {
			return
		}
		timeout.Reset(k.keepAliveInterval)
		select {
		case <-k.pongChan:
			time.Sleep(k.keepAliveInterval / 2)
			continue
		case <-timeout.C:
			return
		}
	}
}

func (k *KeepAlive) pongHandler(string) error {
	select {
	case k.pongChan <- struct{}{}:
	default:
	}
	return nil
}
