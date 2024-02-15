package download

import (
	"demo-storage/internal/app/interfaces"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
)

type Endpoint struct {
	s interfaces.MinioService
}

func New(s interfaces.MinioService) *Endpoint {
	// Создаем endpoint и возвращаем
	return &Endpoint{s: s}
}

func (e *Endpoint) DownloadHandler(ctx echo.Context) error { // Source
	file := ctx.QueryParam("file")
	if file == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide file name")
	}

	res := e.s.DownloadFile(file)
	defer res.Body.Close()
	buff, err := io.ReadAll(res.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error reading objects")
	}

	ctx.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+file)
	return ctx.Blob(http.StatusOK, *res.ContentType, buff)
}
