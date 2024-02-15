package logging

import (
	"demo-storage/internal/config"
	logdoc "github.com/LogDoc-org/logdoc-go-appender/logrus"
	"net"
)

func LDSubsystemInit() (*net.Conn, error) {
	conf := config.GetConfig()
	conn, err := logdoc.Init(
		conf.GetString("ld.proto"),
		conf.GetString("ld.host")+":"+conf.GetString("ld.port"),
		conf.GetString("ld.app"),
	)
	return &conn, err
}
