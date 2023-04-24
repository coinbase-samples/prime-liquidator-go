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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coinbase-samples/prime-liquidator-go/prime"
	"github.com/google/uuid"
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

func RunLiquidator(config AppConfig) {

	l := newLiquidator(config)

	l.monitor()
}

func newLiquidator(config AppConfig) (l *Liquidator) {

	l = &Liquidator{
		config:           config,
		toConvertSymbols: make(map[string]bool),
	}

	for _, s := range config.ConvertSymbols {
		l.toConvertSymbols[s] = true
	}

	l.ordersCache = ttlcache.NewCache()
	l.ordersCache.SetTTL(config.TwapDuration)
	l.ordersCache.SetCacheSizeLimit(1000)

	return
}

type Liquidator struct {
	config           AppConfig
	toConvertSymbols map[string]bool
	balances         []*prime.AssetBalances
	products         ProductLookup
	wallets          WalletLookup
	ordersCache      *ttlcache.Cache
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

	l.wallets, err = l.describeTradingWallets()
	if err != nil {
		return
	}

	l.products, err = l.describeProducts()
	if err != nil {
		return
	}

	l.balances, err = l.describeTradingBalances()
	if err != nil {
		return
	}

	return
}

func (l Liquidator) describeTradingBalances() ([]*prime.AssetBalances, error) {

	ctx, cancel := context.WithTimeout(context.Background(), l.config.PrimeCallTimeout)
	defer cancel()

	response, err := prime.DescribeBalances(
		ctx,
		&prime.DescribeBalancesRequest{PortfolioId: l.config.PortfolioId},
	)

	if err != nil {
		return nil, err
	}

	return response.Balances, nil
}

func (l Liquidator) describeProducts() (ProductLookup, error) {

	ctx, cancel := context.WithTimeout(context.Background(), l.config.PrimeCallTimeout)
	defer cancel()

	products := make(map[string]*prime.Product)

	var cursor string

	for {

		request := &prime.DescribeProductsRequest{
			PortfolioId:    l.config.PortfolioId,
			IteratorParams: &prime.IteratorParams{Cursor: cursor},
		}

		response, err := prime.DescribeProducts(ctx, request)
		if err != nil {
			return nil, err
		}

		for _, p := range response.Products {
			products[p.Id] = p
		}

		if !response.HasNext() {
			break
		}

		cursor = response.Pagination.NextCursor
	}

	return products, nil
}

func (l Liquidator) describeTradingWallets() (WalletLookup, error) {

	ctx, cancel := context.WithTimeout(context.Background(), l.config.PrimeCallTimeout)
	defer cancel()

	var cursor string

	wallets := make(map[string]*prime.Wallet)

	for {

		request := &prime.DescribeWalletsRequest{
			PortfolioId: l.config.PortfolioId,
			Type:        prime.WalletTypeTrading,
			IteratorParams: &prime.IteratorParams{
				Cursor: cursor,
			},
		}

		response, err := prime.DescribeWallets(ctx, request)
		if err != nil {
			return wallets, err
		}

		for _, wallet := range response.Wallets {
			wallets[wallet.Symbol] = wallet
		}

		if !response.HasNext() {
			break
		}

		cursor = response.Pagination.NextCursor
	}

	return wallets, nil
}

func (l *Liquidator) createConversion(
	sourceWallet,
	destinationWallet *prime.Wallet,
	amount decimal.Decimal,
) error {

	ctx, cancel := context.WithTimeout(context.Background(), l.config.PrimeCallTimeout)
	defer cancel()

	round := amount.RoundFloor(2)

	if round.IsZero() {
		return nil
	}

	log.Infof("converting %s to %s - amount: %v", sourceWallet.Symbol, destinationWallet.Symbol, round)

	request := &prime.CreateConversionRequest{
		PortfolioId:         l.config.PortfolioId,
		SourceWalletId:      sourceWallet.Id,
		DestinationWalletId: destinationWallet.Id,
		SourceSymbol:        strings.ToUpper(sourceWallet.Symbol),
		DestinationSymbol:   strings.ToUpper(destinationWallet.Symbol),
		Amount:              round.String(),
		IdempotencyId:       uuid.New().String(),
	}

	response, err := prime.CreateConversion(ctx, request)
	if err != nil {
		return err
	}

	log.Infof(
		"convert request submitted - %s to %s - amount: %v - activity id: %s",
		sourceWallet.Symbol,
		destinationWallet.Symbol,
		round,
		response.ActivityId,
	)

	return nil
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

func (l Liquidator) createOrder(
	productId string,
	value,
	orderSize decimal.Decimal,
	asset *prime.AssetBalances,
	limitPrice decimal.Decimal,
	duration time.Duration,
	clientOrderId string,
) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), l.config.PrimeCallTimeout)
	defer cancel()

	log.Infof("create order request - asset: %s - balance: %s - value: %v - order size: %v", asset.Symbol, asset.Amount, value, orderSize)

	request := l.createOrderRequest(
		productId,
		value,
		orderSize,
		asset,
		limitPrice,
		duration,
		clientOrderId,
	)

	response, err := prime.CreateOrder(ctx, request)
	if err != nil {
		return "", fmt.Errorf(
			"unable to create order - client order id: %s - symbol: %s - size: %v - err: %w",
			clientOrderId,
			asset.Symbol,
			orderSize,
			err,
		)
	}

	log.Infof("new order created - id: %s", response.OrderId)

	return response.OrderId, nil
}

func (l Liquidator) createOrderRequest(
	productId string,
	value,
	orderSize decimal.Decimal,
	asset *prime.AssetBalances,
	limitPrice decimal.Decimal,
	duration time.Duration,
	clientOrderId string,
) *prime.CreateOrderRequest {

	startTime := time.Now()
	endTime := startTime.Add(duration)

	return &prime.CreateOrderRequest{
		PortfolioId:   l.config.PortfolioId,
		ProductId:     productId,
		Side:          prime.OrderSideSell,
		Type:          prime.OrderTypeTwap,
		TimeInForce:   prime.TimeInForceGoodUntilTime,
		ClientOrderId: clientOrderId,
		BaseQuantity:  orderSize.String(),
		LimitPrice:    limitPrice.String(),
		StartTime:     startTime.Format("2006-01-02T15:04:05Z"),
		ExpiryTime:    endTime.Format("2006-01-02T15:04:05Z"),
	}
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

		if err := l.createConversion(stablecoinWallet, fiatWallet, amount); err != nil {
			return err
		}

		return nil
	}

	price, err := currentExchangeProductPrice(
		strings.ToUpper(
			fmt.Sprintf("%s-%s", asset.Symbol, l.config.FiatCurrencySymbol),
		),
		l.config.PrimeCallTimeout,
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

	if orderId, err := l.createOrder(
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
