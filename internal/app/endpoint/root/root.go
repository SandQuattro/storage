package root

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Endpoint struct{}

func New() *Endpoint {
	// Создаем endpoint и возвращаем
	return &Endpoint{}
}

func (e *Endpoint) RootHandler(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "Сервис storage online!")
}
