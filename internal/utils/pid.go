package utils

import (
	"os"
	"strconv"

	logdoc "github.com/SandQuattro/logdoc-go-appender/logrus"
)

func CreatePID() {
	logger := logdoc.GetLogger()
	// Сохраним id запущенного процесса в файл
	pid := os.Getpid()
	err := os.WriteFile("RUNNING_PID", []byte(strconv.Itoa(pid)), 0o600)
	if err != nil {
		logger.Fatal("Error writing PID to file. Exiting...")
	}
	logger.Info("Service RUNNING PID created")
}
