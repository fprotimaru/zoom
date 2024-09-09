package handlers

import "github.com/gorilla/websocket"

type Call interface {
	AddWebSocketClient(id string, conn *websocket.Conn)
}

type Handler struct {
	call Call
}

func NewHandler(call Call) *Handler {
	return &Handler{
		call: call,
	}
}
