package wsconn

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"zoom/internal/entity"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 4096 * 2
)

type MessageHandlerFunc func(id string, msg *entity.WebSocketMessage)
type DisconnectHandlerFunc func(id string)

type Config struct {
	messageCB    MessageHandlerFunc
	disconnectCB DisconnectHandlerFunc
}

type WSConn struct {
	cfg  *Config
	id   string
	mu   sync.RWMutex
	conn *websocket.Conn

	onceClose sync.Once

	callDisconnectCB bool

	messageChan chan []byte
}

func NewWSConn(id string, conn *websocket.Conn) *WSConn {
	client := &WSConn{
		cfg:         new(Config),
		id:          id,
		mu:          sync.RWMutex{},
		conn:        conn,
		onceClose:   sync.Once{},
		messageChan: make(chan []byte),
	}

	client.conn.SetCloseHandler(client.closed)

	go client.read()
	go client.write()

	return client
}

func (w *WSConn) GetID() string {
	return w.id
}

func (w *WSConn) Close(callDisconnectCB bool) {
	w.callDisconnectCB = callDisconnectCB
	w.close()
}

func (w *WSConn) SetMessageCB(cb MessageHandlerFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cfg == nil {
		return
	}

	w.cfg.messageCB = cb
}

func (w *WSConn) SetDisconnectCB(cb DisconnectHandlerFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cfg == nil {
		return
	}

	w.cfg.disconnectCB = cb
}

func (w *WSConn) SendMessage(data any) {
	if data == nil {
		return
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return
	}

	w.sendMessage(raw)
}

func (w *WSConn) sendMessage(body []byte) {
	w.messageChan <- body
}

func (w *WSConn) read() {
	defer func() {
		w.callDisconnectCB = true
		w.close()
	}()

	w.conn.SetReadLimit(maxMessageSize)
	if err := w.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return
	}
	w.conn.SetPongHandler(func(s string) error {
		return w.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var msg entity.WebSocketMessage
		if err := w.conn.ReadJSON(&msg); err != nil {
			log.Error(err)
			break
		}

		if w.cfg.messageCB != nil {
			w.cfg.messageCB(w.id, &msg)
		}
	}
}

func (w *WSConn) write() {
	defer func() {
		w.callDisconnectCB = true
		w.close()
	}()

	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		w.close()
	}()

	for {
		select {
		case <-ticker.C:
			if err := w.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := w.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case msg := <-w.messageChan:
			if err := w.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := w.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}
}

func (w *WSConn) closed(code int, text string) error {
	w.callDisconnectCB = true
	w.close()

	return nil
}

func (w *WSConn) close() {
	fmt.Println("closing")
	w.onceClose.Do(func() {
		err := w.conn.Close()
		if err != nil {

		}
		if w.callDisconnectCB {
			if w.cfg != nil {
				w.cfg.disconnectCB(w.id)
			}
		}
	})
}
