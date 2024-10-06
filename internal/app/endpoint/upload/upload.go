package upload

import (
	"demo-storage/internal/app/interfaces"
	"demo-storage/internal/app/structs"
	"fmt"
	"github.com/gurkankaymak/hocon"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/url"
)

type Endpoint struct {
	config *hocon.Config
	s      interfaces.MinioService
}

func New(s interfaces.MinioService, config *hocon.Config) *Endpoint {
	// Создаем endpoint и возвращаем
	return &Endpoint{s: s, config: config}
}

func (e *Endpoint) UploadHandler(ctx echo.Context) error { // Source
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, err)
	}

	//hostname, err := os.Hostname()
	//if err != nil {
	//	return ctx.JSON(http.StatusBadRequest, err)
	//}
	proto := e.config.GetString("server.proto")
	host := e.config.GetString("server.address")
	filePath := fmt.Sprintf("%s://%s/download?file=%s", proto, host, url.QueryEscape(fileHeader.Filename))
	go e.s.UploadFile(fileHeader, filePath)

	u := &structs.Response{
		FilePath: filePath,
		Result:   fmt.Sprintf("Запущена загрузка файла: %s. Для проверки статуса: GET /status?file=<file name>", fileHeader.Filename),
	}
	return ctx.JSON(http.StatusOK, u)
}
