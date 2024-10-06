package logging

import (
	"net"

	"demo-storage/internal/config"
	logdoc "github.com/SandQuattro/logdoc-go-appender/logrus"
)

func LDSubsystemInit() (*net.Conn, error) {
	conf := config.GetConfig()
	conn, err := logdoc.Init(
		conf.GetString("ld.proto"),
		conf.GetString("ld.host")+":"+conf.GetString("ld.port"),
		conf.GetString("ld.app"),
		logdoc.TEXT,
	)
	return &conn, err
}
