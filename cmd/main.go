package main

import (
	"zoom/internal/handlers"
	"zoom/internal/service/call"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()
	//e.Use(middleware.Logger())

	callSvc := call.NewCall()

	h := handlers.NewHandler(callSvc)

	e.Static("/", "template")
	e.GET("/ws/:id", h.WebSocket)

	if err := e.Start(":8000"); err != nil {
		log.Fatalf("echo start: %v", err)
	}
}
