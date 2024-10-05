package app

import (
	"errors"
	"net/http"
	"strings"

	"demo-storage/internal/app/endpoint/buckets"
	"demo-storage/internal/app/endpoint/download"
	"demo-storage/internal/app/endpoint/objects"
	"demo-storage/internal/app/endpoint/root"
	"demo-storage/internal/app/endpoint/status"
	wsupload "demo-storage/internal/app/endpoint/upload/multipartws"
	"demo-storage/internal/app/mv"
	minio "demo-storage/internal/app/service"

	logdoc "github.com/LogDoc-org/logdoc-go-appender/logrus"
	"github.com/gurkankaymak/hocon"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type App struct {
	port     string
	access   string
	secret   string
	db       *sqlx.DB
	config   *hocon.Config
	Echo     *echo.Echo
	root     *root.Endpoint
	status   *status.Endpoint
	wsupload *wsupload.Endpoint
	download *download.Endpoint
	buckets  *buckets.Endpoint
	objects  *objects.Endpoint
	s        *minio.MinioService
}

func New(config *hocon.Config, port string, access string, secret string, db *sqlx.DB) (*App, error) {
	a := App{port: port, config: config, access: access, secret: secret, db: db}

	a.s = minio.New(config, access, secret, db)

	a.root = root.New()
	a.status = status.New(db)
	a.download = download.New(a.s)
	a.buckets = buckets.New(a.s)
	a.objects = objects.New(a.s)

	// multipart upload using websockets
	a.wsupload = wsupload.New(a.s, config)

	// Echo instance
	a.Echo = echo.New()

	// TODO add debug
	pprof.Register(a.Echo)

	// Global Endpoints Middleware
	// Вызов перед каждым обработчиком
	// В них может быть логгирование,
	// поверка токенов, ролей, прав и многое другое
	a.Echo.Use(middleware.Logger())
	a.Echo.Use(middleware.Recover())
	a.Echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	}))

	// Metrics middleware
	a.Echo.Use(echoprometheus.NewMiddleware("storage_demo_storage"))
	a.Echo.GET("/storage/metrics", echoprometheus.NewHandler())

	a.Echo.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	// Body dump mv captures the request and response payload and calls the registered handler.
	// Generally used for debugging/logging purpose. Avoid using it if your request/response payload is huge e.g.
	// file upload/download

	a.Echo.Use(middleware.BodyDumpWithConfig(
		middleware.BodyDumpConfig{
			Skipper: func(c echo.Context) bool {
				return strings.Contains(c.Request().URL.Path, "/download") ||
					strings.Contains(c.Request().URL.Path, "/storage/upload") ||
					strings.Contains(c.Request().URL.Path, "/ws/upload")
			},
			Handler: func(c echo.Context, reqBody, resBody []byte) {
				c.Logger().Debug("Response: " + string(resBody))
			},
		}))

	// Routes
	a.Echo.GET("/", a.root.RootHandler)
	a.Echo.GET("/status", a.status.StatusHandler)
	a.Echo.GET("/buckets", a.buckets.BucketsHandler, mv.HeaderCheck(config))
	a.Echo.GET("/objects/list", a.objects.ObjectsHandler, mv.HeaderCheck(config))
	a.Echo.GET("/download", a.download.DownloadHandler)
	a.Echo.GET("/ws/upload", a.wsupload.WebSocketUploadHandler)

	return &a, nil
}

func (a *App) Run() error {
	logger := logdoc.GetLogger()
	// Start server
	err := a.Echo.Start(":" + a.port)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Warn("Ошибка запуска сервера:", err)
	}
	return nil
}
