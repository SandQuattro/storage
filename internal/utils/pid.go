package utils

import (
	logdoc "github.com/LogDoc-org/logdoc-go-appender/logrus"
	"os"
	"strconv"
)

func CreatePID() {
	logger := logdoc.GetLogger()
	// Сохраним id запущенного процесса в файл
	pid := os.Getpid()
	err := os.WriteFile("RUNNING_PID", []byte(strconv.Itoa(pid)), 0600)
	if err != nil {
		logger.Fatal("Error writing PID to file. Exiting...")
	}
	logger.Info("Service RUNNING PID created")
}
