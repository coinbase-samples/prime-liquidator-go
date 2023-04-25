/**
 * Copyright 2023-present Coinbase Global, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coinbase-samples/prime-liquidator-go/config"
	"github.com/coinbase-samples/prime-liquidator-go/monitor"
	"github.com/coinbase-samples/prime-liquidator-go/prime"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

func main() {

	run := make(chan os.Signal, 1)
	signal.Notify(run, os.Interrupt, syscall.SIGTERM)

	config.LogInit()

	if err := os.Setenv("TZ", "UTC"); err != nil {
		log.Fatalf("Cannot set time zone: UTC: %v", err)
	}

	log.Info("starting daemon")

	credentials, err := prime.InitCredentials()
	if err != nil {
		log.Fatalf("unable to init prime credentials: %v", err)
	}

	log.Warn("watch for crypto assets in hot wallets/trading and sell")

	config := config.AppConfig{
		PortfolioId:            credentials.PortfolioId,
		FiatCurrencySymbol:     "USD",
		TwapDuration:           6 * time.Minute, // Change to 60 before release
		ConvertSymbols:         []string{"usdc"},
		PrimeCallTimeout:       30 * time.Second,
		TwapMaxDiscountPercent: decimal.NewFromFloat32(0.1),
		OrdersCacheSize:        1000,
		StablecoinFiatDigits:   2,
	}

	go monitor.RunLiquidator(config)

	<-run
}
