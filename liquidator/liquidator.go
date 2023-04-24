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

package liquidator

import (
	"fmt"
	"strings"
	"time"

	"github.com/coinbase-samples/prime-liquidator-go/prime"
	"github.com/jellydator/ttlcache/v2"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type AppConfig struct {
	PortfolioId            string
	FiatCurrencySymbol     string
	TwapDuration           time.Duration
	ConvertSymbols         []string
	PrimeCallTimeout       time.Duration
	TwapMaxDiscountPercent decimal.Decimal
}

type ProductLookup map[string]*prime.Product

type WalletLookup map[string]*prime.Wallet

type Liquidator struct {
	config           AppConfig
	toConvertSymbols map[string]bool
	balances         []*prime.AssetBalances
	products         ProductLookup
	wallets          WalletLookup
	ordersCache      *ttlcache.Cache
	api              ApiCall
}

func RunLiquidator(config AppConfig) {

	l := newLiquidator(config)

	l.monitor()
}

func newLiquidator(config AppConfig) (l *Liquidator) {

	l = &Liquidator{
		config:           config,
		toConvertSymbols: make(map[string]bool),
		api:              ApiCall{config: config},
	}

	for _, s := range config.ConvertSymbols {
		l.toConvertSymbols[s] = true
	}

	l.ordersCache = ttlcache.NewCache()
	l.ordersCache.SetTTL(config.TwapDuration)
	l.ordersCache.SetCacheSizeLimit(1000)

	return
}

func (l *Liquidator) monitor() {

	for {

		if err := l.describeCurrentState(); err != nil {
			log.Error(err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, asset := range l.balances {
			if err := l.processAsset(asset); err != nil {
				log.Error(err)
			}
			time.Sleep(1 * time.Second)
		}

		time.Sleep(5 * time.Second)
	}
}

func (l *Liquidator) describeCurrentState() (err error) {

	l.wallets, err = l.api.describeTradingWallets()
	if err != nil {
		return
	}

	l.products, err = l.api.describeProducts()
	if err != nil {
		return
	}

	l.balances, err = l.api.describeTradingBalances()
	if err != nil {
		return
	}

	return
}

func (l Liquidator) calculateTwapLimitPrice(
	productId string,
	price decimal.Decimal,
) (limitPrice decimal.Decimal, err error) {

	var (
		quoteIncrement decimal.Decimal
	)

	product, found := l.products[productId]
	if !found {
		err = fmt.Errorf("Unknown product id: %s", productId)
		return
	}

	if quoteIncrement, err = product.QuoteIncrementNum(); err != nil {
		return
	}

	maxDiscount := price.Mul(l.config.TwapMaxDiscountPercent)

	limitPrice = l.adjustTwapLimitPrice(price.Sub(maxDiscount), quoteIncrement)
	return
}

func (l *Liquidator) processAsset(asset *prime.AssetBalances) error {

	if asset.IsFiat() {
		return nil
	}

	amount, err := asset.AmountNum()
	if err != nil {
		return err
	}

	if amount.IsZero() {
		return nil
	}

	// Check for stablecoins that need to be converted
	if _, found := l.toConvertSymbols[asset.Symbol]; found {

		fiatWallet, found := l.wallets[l.config.FiatCurrencySymbol]

		if !found {
			return fmt.Errorf("fiat wallet not found: %s", l.config.FiatCurrencySymbol)
		}

		stablecoinWallet, found := l.wallets[strings.ToUpper(asset.Symbol)]

		if !found {
			return fmt.Errorf("stablecoin wallet not found: %s", asset.Symbol)
		}

		if err := l.api.createConversion(stablecoinWallet, fiatWallet, amount); err != nil {
			return err
		}

		return nil
	}

	price, err := l.api.currentExchangeProductPrice(
		strings.ToUpper(
			fmt.Sprintf("%s-%s", asset.Symbol, l.config.FiatCurrencySymbol),
		),
	)

	if err != nil {
		return fmt.Errorf("cannot get exchange price: %s - err: %w", asset.Symbol, err)
	}

	productId := fmt.Sprintf("%s-%s", strings.ToUpper(asset.Symbol), l.config.FiatCurrencySymbol)

	value := amount.Mul(price)

	if value.IsZero() {
		return nil
	}

	holds, err := asset.HoldsNum()
	if err != nil {
		return err
	}

	orderSize, err := l.calculateOrderSize(productId, amount, holds)
	if err != nil {
		return err
	}

	if orderSize.IsZero() {
		return nil
	}

	limitPrice, err := l.calculateTwapLimitPrice(productId, price)
	if err != nil {
		return err
	}

	clientOrderId := prime.GenerateUniqueId(
		productId,
		prime.OrderSideSell,
		prime.OrderTypeTwap,
		prime.TimeInForceGoodUntilTime,
		orderSize.String(),
	)

	if _, exists := l.ordersCache.Get(clientOrderId); exists == nil {
		return nil
	}

	if orderId, err := l.api.createOrder(
		productId,
		value,
		orderSize,
		asset,
		limitPrice,
		l.config.TwapDuration,
		clientOrderId,
	); err != nil {
		return err
	} else {
		l.ordersCache.Set(clientOrderId, orderId)
	}

	return nil
}

func (l Liquidator) calculateOrderSize(
	productId string,
	amount decimal.Decimal,
	holds decimal.Decimal,
) (orderSize decimal.Decimal, err error) {

	product, found := l.products[productId]
	if !found {
		err = fmt.Errorf("Unknown product id: %s", productId)
		return
	}

	orderSize, err = prime.CalculateOrderSize(product, amount, holds)

	return
}

func (l Liquidator) adjustTwapLimitPrice(price, quoteIncrement decimal.Decimal) decimal.Decimal {
	quo, rem := price.QuoRem(quoteIncrement, 0)

	if rem.IsZero() {
		return price
	}

	return quo.Floor().Mul(quoteIncrement)
}
