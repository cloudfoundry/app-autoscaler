package testhelpers

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebsocketHandler struct {
	messages     chan []byte
	messageTypes chan int
	keepAlive    time.Duration
	lock         *sync.Mutex
}

func NewWebsocketHandler(m chan []byte, mt chan int, keepAlive time.Duration) *WebsocketHandler {
	return &WebsocketHandler{
		messages:     m,
		messageTypes: mt,
		keepAlive:    keepAlive,
		lock:         &sync.Mutex{},
	}
}

func (h *WebsocketHandler) ServeWebsocket(rw http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	ws, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		log.Printf("websocket handler: Not a websocket handshake: %s", err)
		return
	}
	defer ws.Close()

	closeCode, closeMessage := h.runWebsocketUntilClosed(ws)
	ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, closeMessage), time.Time{})
}

func (h *WebsocketHandler) runWebsocketUntilClosed(ws *websocket.Conn) (closeCode int, closeMessage string) {
	keepAliveExpired := make(chan struct{})
	clientWentAway := make(chan struct{})

	go func() {
		NewKeepAlive(h.lock, ws, h.keepAlive).Run()
		close(keepAliveExpired)
	}()
	closeCode = websocket.CloseNormalClosure
	closeMessage = ""
	for {
		select {
		case <-clientWentAway:
			h.messageTypes <- websocket.CloseMessage
			return
		case <-keepAliveExpired:
			closeCode = websocket.ClosePolicyViolation
			closeMessage = "Client did not respond to ping before keep-alive timeout expired."
			return
		default:
			_, message, err := ws.ReadMessage()
			if err != nil {
				close(clientWentAway)
			} else {
				h.messages <- message

			}
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
