package buckets

import (
	"net/http"

	"demo-storage/internal/app/interfaces"
	"github.com/labstack/echo/v4"
)

type Endpoint struct {
	s interfaces.MinioService
}

func New(s interfaces.MinioService) *Endpoint {
	// Создаем endpoint и возвращаем
	return &Endpoint{s: s}
}

func (e *Endpoint) BucketsHandler(ctx echo.Context) error {
	res := e.s.ListBuckets()
	return ctx.JSON(http.StatusOK, res)
}
