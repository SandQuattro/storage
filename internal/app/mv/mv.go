package mv

import (
	jwtservice "demo-storage/internal/app/security"
	logdoc "github.com/LogDoc-org/logdoc-go-appender/logrus"
	"github.com/gurkankaymak/hocon"
	"github.com/labstack/echo/v4"
	"net/http"
)

const AUTHORIZATION = "Authorization"

func HeaderCheck(config *hocon.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			logger := logdoc.GetLogger()
			logger.Debug("Header Check Middleware executed")

			token := ctx.Request().Header.Get(AUTHORIZATION)
			isValid, err := jwtservice.ValidateToken(token, config)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
			}

			if token != "" && isValid {
				err := next(ctx)
				if err != nil {
					logger.Error("Authorization header error")
					return err
				}
				return nil
			}
			return echo.NewHTTPError(http.StatusUnauthorized, "Please provide valid credentials")
		}
	}
}
