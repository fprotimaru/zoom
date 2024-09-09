package handlers

import (
	"net/http"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func (h *Handler) WebSocket(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	userID := c.Param("id")

	log.Debugf("user: %s connected!", userID)

	h.call.AddWebSocketClient(userID, ws)
	return nil
}
