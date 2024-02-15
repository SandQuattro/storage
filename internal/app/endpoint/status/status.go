package status

import (
	"demo-storage/internal/app/interfaces"
	"demo-storage/internal/app/repository"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"net/http"
)

type Endpoint struct {
	r interfaces.UserRepository
}

func New(db *sqlx.DB) *Endpoint {
	// Создаем endpoint и возвращаем
	r := repository.New(db)
	return &Endpoint{r: r}
}

func (e *Endpoint) StatusHandler(ctx echo.Context) error {
	name := ctx.QueryParam("file")
	res := e.r.FindFileByName(name)
	if res == nil || res.Name == "" {
		return echo.NewHTTPError(http.StatusNotFound, "File not found with name ", name)
	}
	return ctx.JSON(http.StatusOK, res)
}
