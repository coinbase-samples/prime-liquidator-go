package config

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func LogInit() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)
	logLevel, _ := log.ParseLevel("info")
	log.SetLevel(logLevel)
	log.SetOutput(os.Stdout)
}
