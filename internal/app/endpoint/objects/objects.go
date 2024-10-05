package objects

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

func (e *Endpoint) ObjectsHandler(ctx echo.Context) error { // Source
	bucket := ctx.QueryParam("bucket")
	if bucket == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide bucket name")
	}

	res := e.s.ListObjects(bucket)
	if res == nil {
		return ctx.String(http.StatusInternalServerError, "Ошибка получения данных")
	} else {
		return ctx.JSON(http.StatusOK, res)
	}
}
