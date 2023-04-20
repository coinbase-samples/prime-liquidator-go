package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.cbhq.net/ryan-nitz/prime-liquidator/config"
	"github.cbhq.net/ryan-nitz/prime-liquidator/liquidator"
	"github.cbhq.net/ryan-nitz/prime-liquidator/prime"
	log "github.com/sirupsen/logrus"
)

func main() {

	run := make(chan os.Signal, 1)
	signal.Notify(run, os.Interrupt, syscall.SIGTERM)

	config.LogInit()

	if err := os.Setenv("TZ", "UTC"); err != nil {
		log.Fatalf("Cannot set time zone: UTC: %v", err)
	}

	log.Info("Starting server")

	if _, err := prime.InitCredentials(); err != nil {
		log.Fatalf("Unable to init prime credentials: %v", err)
	}

	log.Info("Watch for digital assets and convert to fiat")

	go liquidator.ConvertToFiat()

	<-run
}
