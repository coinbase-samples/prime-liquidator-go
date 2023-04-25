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

package caller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coinbase-samples/prime-liquidator-go/config"
	"github.com/coinbase-samples/prime-liquidator-go/exchange"
	"github.com/coinbase-samples/prime-liquidator-go/prime"
	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v2"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type apiCall struct {
	config      config.AppConfig
	ordersCache *ttlcache.Cache
}

func NewCaller(config config.AppConfig) Caller {

	ordersCache := ttlcache.NewCache()
	ordersCache.SetTTL(config.TwapDuration)
	ordersCache.SetCacheSizeLimit(config.OrdersCacheSize)

	return apiCall{
		config:      config,
		ordersCache: ordersCache,
	}
}

func (ac apiCall) PrimeDescribeTradingWallets() (WalletLookup, error) {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout)
	defer cancel()

	var cursor string

	wallets := make(WalletLookup)

	for {

		request := &prime.DescribeWalletsRequest{
			PortfolioId: ac.config.PortfolioId,
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
			wallets.Add(wallet)
		}

		if !response.HasNext() {
			break
		}

		cursor = response.Pagination.NextCursor
	}

	return wallets, nil
}

func (ac apiCall) PrimeDescribeProducts() (ProductLookup, error) {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout)
	defer cancel()

	products := make(ProductLookup)

	var cursor string

	for {

		request := &prime.DescribeProductsRequest{
			PortfolioId:    ac.config.PortfolioId,
			IteratorParams: &prime.IteratorParams{Cursor: cursor},
		}

		response, err := prime.DescribeProducts(ctx, request)
		if err != nil {
			return nil, err
		}

		for _, p := range response.Products {
			products.Add(p)
		}

		if !response.HasNext() {
			break
		}

		cursor = response.Pagination.NextCursor
	}

	return products, nil
}

func (ac apiCall) PrimeCalculateOrderSize(
	product *prime.Product,
	amount,
	holds decimal.Decimal,
) (orderSize decimal.Decimal, err error) {
	orderSize, err = prime.CalculateOrderSize(product, amount, holds)
	if err != nil {
		err = fmt.Errorf(
			"cannot calculator order size - product: %s - amount: %v - holds: %v - err: %v",
			product.Id,
			amount,
			holds,
			err,
		)
	}
	return
}

func (ac apiCall) PrimeDescribeTradingBalances() ([]*prime.AssetBalances, error) {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout)
	defer cancel()

	response, err := prime.DescribeBalances(
		ctx,
		&prime.DescribeBalancesRequest{
			PortfolioId: ac.config.PortfolioId,
			Type:        prime.BalanceTypeTrading,
		},
	)

	if err != nil {
		return nil, err
	}

	return response.Balances, nil
}

func (ac apiCall) PrimeCreateConversion(
	sourceWallet,
	destinationWallet *prime.Wallet,
	amount decimal.Decimal,
) error {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout)
	defer cancel()

	round := amount.RoundFloor(ac.config.StablecoinFiatDigits)

	if round.IsZero() {
		return nil
	}

	log.Infof("converting %s to %s - amount: %v", sourceWallet.Symbol, destinationWallet.Symbol, round)

	request := &prime.CreateConversionRequest{
		PortfolioId:         ac.config.PortfolioId,
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

func (ac apiCall) PrimeCreateTwapOrder(
	productId string,
	value,
	orderSize,
	limitPrice decimal.Decimal,
	asset *prime.AssetBalances,
) error {

	holds, err := asset.HoldsNum()
	if err != nil {
		return err
	}

	clientOrderId := prime.GenerateUniqueId(
		productId,
		prime.OrderSideSell,
		prime.OrderTypeTwap,
		prime.TimeInForceGoodUntilTime,
		orderSize.String(),
		holds.String(),
	)

	if _, exists := ac.ordersCache.Get(clientOrderId); exists == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout)
	defer cancel()

	log.Infof("create order request - asset: %s - balance: %s - value: %v - order size: %v", asset.Symbol, asset.Amount, value, orderSize)

	request := ac.createOrderRequest(
		productId,
		value,
		orderSize,
		asset,
		limitPrice,
		ac.config.TwapDuration,
		clientOrderId,
	)

	response, err := prime.CreateOrder(ctx, request)
	if err != nil {
		return fmt.Errorf(
			"unable to create order - client order id: %s - symbol: %s - size: %v - err: %w",
			clientOrderId,
			asset.Symbol,
			orderSize,
			err,
		)
	}

	log.Infof("order created - id: %s - client order id: %s", response.OrderId, clientOrderId)

	ac.ordersCache.Set(clientOrderId, response.OrderId)

	return nil
}

func (ac apiCall) createOrderRequest(
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
		PortfolioId:   ac.config.PortfolioId,
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

func (ac apiCall) ExchangeCurrentProductPrice(productId string) (decimal.Decimal, error) {
	return exchange.CurrentProductPrice(productId, ac.config.PrimeCallTimeout)
}
