package main

import (
	"demo-storage/internal/pkg/db"
	"demo-storage/internal/pkg/logging"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"demo-storage/internal/config"
	"demo-storage/internal/pkg/app"
	"demo-storage/internal/utils"
	"demo-storage/internal/utils/gs"

	"github.com/SandQuattro/logdoc-go-appender/common"
	logdoc "github.com/SandQuattro/logdoc-go-appender/logrus"
	"github.com/jmoiron/sqlx"
)

func main() {
	// Первым делом считаем аргументы командной строки
	// Читаем параметры командной строки
	confFile := flag.String("config", "conf/application.conf", "-config=<config file name>")
	port := flag.String("port", "", "-port=<service port>")
	flag.Parse()

	if *port == "" {
		log.Fatal("Empty service port. Exiting...")
	}

	access := os.Getenv("MINIO_ACCESS")
	secret := os.Getenv("MINIO_SECRET")
	if access == "" || secret == "" {
		log.Fatal("Empty access or secret key. Exiting...")
	}

	// и подгрузим конфиг
	config.MustConfig(*confFile)
	conf := config.GetConfig()

	// Создаем подсистему логгирования LogDoc
	conn, err := logging.LDSubsystemInit()
	logger := logdoc.GetLogger()
	if err == nil {
		logger.Info(fmt.Sprintf(
			"LogDoc subsystem initialized successfully@@source=%s:%d",
			common.GetSourceName(runtime.Caller(0)), // фреймы не скипаем, не exception
			common.GetSourceLineNum(runtime.Caller(0)),
		))
	}

	c := *conn
	if c != nil {
		defer c.Close()
	} else {
		logger.Error("Error LogDoc subsystem initialization")
	}

	utils.CreatePID()
	defer func() {
		err := os.Remove("RUNNING_PID")
		if err != nil {
			logger.Fatal("Error removing PID file. Exiting...")
		}
	}()

	// Коннектимся к базе
	dbPass := os.Getenv("PGPASS")
	if dbPass == "" {
		logger.Fatal("db password is empty")
	}

	// Коннектимся к базе
	d := db.Connect(conf, dbPass)
	defer func(d *sqlx.DB) {
		err := d.Close()
		if err != nil {
			logger.Fatal(err)
		}
	}(d)
	logger.Info(">> DATABASE CONNECTION SUCCESSFUL")

	// Создадим приложение
	a, err := app.New(conf, *port, access, secret, d)
	if err != nil {
		logger.Error("Ошибка создания приложения")
	}

	go func() {
		// и запустим приложение (веб сервер)
		logger.Debug(fmt.Sprintf(">> RUNNING SERVER ON PORT: %s", *port))
		err = a.Run()
		if err != nil {
			logger.Fatal(err)
		}
	}()
	gs.GracefulShutdown(a)
}
