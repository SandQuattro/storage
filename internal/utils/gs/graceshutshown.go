package gs

import (
	"context"
	"demo-storage/internal/pkg/app"
	logdoc "github.com/LogDoc-org/logdoc-go-appender/logrus"
	"os"
	"os/signal"
)

func GraceShutdown(app *app.App) {
	logger := logdoc.GetLogger()
	// Используем буферизированный канал, как рекомендовано внутри signal.Notify функции
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// Блокируемся и ожидаем из канала quit - interrupt signal чтобы сделать gracefully shutdown сервака с таймаутом в 10 сек
	<-quit

	// Получили SIGINT (0x2), выполняем grace shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger.Warn("Gracefully shutdown server...")
	if err := app.Echo.Shutdown(ctx); err != nil {
		logger.Error("gracefully shutdown error")
	}
}
