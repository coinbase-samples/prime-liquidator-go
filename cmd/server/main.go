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

	"github.com/coinbase-samples/prime-liquidator-go/config"
	"github.com/coinbase-samples/prime-liquidator-go/monitor"
	"github.com/coinbase/prime-sdk-go/balances"
	"github.com/coinbase/prime-sdk-go/client"
	"github.com/coinbase/prime-sdk-go/credentials"
	"github.com/coinbase/prime-sdk-go/orders"
	"github.com/coinbase/prime-sdk-go/products"
	"github.com/coinbase/prime-sdk-go/transactions"
	"github.com/coinbase/prime-sdk-go/wallets"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {

	run := make(chan os.Signal, 1)
	signal.Notify(run, os.Interrupt, syscall.SIGTERM)

	log := config.LogInit("prime-liquidator")
	zap.ReplaceGlobals(log)
	defer log.Sync()

	log.Info("prime-liquidator", zap.String("state", "starting"))

	_ = godotenv.Load()

	if err := os.Setenv("TZ", "UTC"); err != nil {
		log.Fatal("cannot set time zone: UTC", zap.Error(err))
	}

	creds, err := credentials.ReadEnvCredentials("PRIME_CREDENTIALS")
	if err != nil {
		log.Fatal("cannot init the prime credentials", zap.Error(err))
	}

	appConfig := &config.AppConfig{}

	if err := config.SetupAppConfig(appConfig); err != nil {
		log.Fatal("cannot setup app config", zap.Error(err))
	}

	appConfig.PrimeClient = client.NewRestClient(creds, *appConfig.HttpClient)
	appConfig.Wallets = wallets.NewWalletsService(appConfig.PrimeClient)
	appConfig.Products = products.NewProductsService(appConfig.PrimeClient)
	appConfig.Balances = balances.NewBalancesService(appConfig.PrimeClient)
	appConfig.Orders = orders.NewOrdersService(appConfig.PrimeClient)
	appConfig.Transactions = transactions.NewTransactionsService(appConfig.PrimeClient)

	if appConfig.IsDryRun() {
		log.Warn("dry run enabled: orders and conversions will not be submitted")
	} else {
		log.Warn("dry run disabled: the liquidator will place real orders and conversions")
	}

	log.Info("watch for crypto assets in hot/trading wallets and sell")

	daemon, err := monitor.StartLiquidator(appConfig)
	if err != nil {
		log.Fatal("cannot start liquidator", zap.Error(err))
	}

	log.Info("prime-liquidator", zap.String("state", "started"))

	<-run

	log.Info("prime-liquidator", zap.String("state", "stopping"))

	if err := monitor.StopLiquidator(daemon); err != nil {
		log.Error("process did not stop cleanly", zap.Error(err))
	}

	log.Info("prime-liquidator", zap.String("state", "stopped"))

}