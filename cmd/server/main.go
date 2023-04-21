package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/coinbase-samples/prime-liquidator-go/config"
	"github.com/coinbase-samples/prime-liquidator-go/liquidator"
	"github.com/coinbase-samples/prime-liquidator-go/prime"
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
