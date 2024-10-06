package multipartws

import (
	"demo-storage/internal/app/interfaces"
	logdoc "github.com/SandQuattro/logdoc-go-appender/logrus"
	"github.com/gorilla/websocket"
	"github.com/gurkankaymak/hocon"
	"github.com/labstack/echo/v4"
	"net/http"
	"sync"
	"time"
)

type Endpoint struct {
	config *hocon.Config
	s      interfaces.MinioService
}

func New(s interfaces.MinioService, config *hocon.Config) *Endpoint {
	// Создаем endpoint и возвращаем
	return &Endpoint{s: s, config: config}
}

type UploadStatus struct {
	Code   int    `json:"code,omitempty"`
	Status string `json:"status,omitempty"`
	Pct    *int64 `json:"part,omitempty"` // File processing AFTER uploading is done.
	pct    int64
}

const (
	HandshakeTimeoutSecs = 10
)

func (e *Endpoint) WebSocketUploadHandler(ctx echo.Context) error { // Source
	var err error
	var ws *websocket.Conn

	logger := logdoc.GetLogger()

	// we can read or write into websocket only in one goroutine
	mu := sync.Mutex{}

	logger.Debug("WebSocketUploadHandler > Starting...")

	// Open websocket connection.
	upgrader := websocket.Upgrader{HandshakeTimeout: time.Second * HandshakeTimeoutSecs}

	// A CheckOrigin function should carefully validate the request origin to
	// prevent cross-site request forgery.
	upgrader.CheckOrigin = func(_ *http.Request) bool {
		// logger.Debug("WebSocketUploadHandler > CheckOrigin > Origin:", r.Header.Get("Origin"))
		// allowedOrigins := []string{"http://localhost:3000", e.config.GetString("server.address")}
		// return slices.Contains(allowedOrigins, r.Header.Get("Origin"))
		return true
	}

	ws, err = upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		logger.Debug("Error on open of websocket connection:", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Error on open of websocket connection")
	}
	defer ws.Close()

	e.processingLoop(ws, &mu)

	return nil
}
