package config

import (
	"github.com/gurkankaymak/hocon"
	"log"
)

var config *hocon.Config

func GetConfig() *hocon.Config {
	return config
}

func MustConfig(confFile string) {
	if confFile == "" {
		log.Fatal("Empty configuration file. Exiting...")
	}
	c, e := hocon.ParseResource(confFile)
	if e != nil {
		log.Fatal("Error reading app configuration file. Exiting...")
	}
	config = c
}
