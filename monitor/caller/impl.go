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
	"go.uber.org/zap"

	prime "github.com/coinbase-samples/prime-sdk-go"
	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v2"
	"github.com/shopspring/decimal"
)

type apiCall struct {
	config      *config.AppConfig
	ordersCache *ttlcache.Cache
	portfolioId string
}

func NewCaller(config *config.AppConfig) Caller {

	ordersCache := ttlcache.NewCache()
	ordersCache.SetTTL(config.TwapDuration())
	ordersCache.SetCacheSizeLimit(config.OrdersCacheSize())

	return apiCall{
		config:      config,
		ordersCache: ordersCache,
		portfolioId: config.PrimeClient.Credentials.PortfolioId,
	}
}

func (ac apiCall) PrimeDescribeTradingWallets() (WalletLookup, error) {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout())
	defer cancel()

	var cursor string

	wallets := make(WalletLookup)

	for {

		request := &prime.ListWalletsRequest{
			PortfolioId: ac.portfolioId,
			Type:        prime.WalletTypeTrading,
			Pagination: &prime.PaginationParams{
				Cursor: cursor,
			},
		}

		response, err := ac.config.PrimeClient.ListWallets(ctx, request)
		if err != nil {
			return wallets, err
		}

		for _, wallet := range response.Wallets {
			wallets.Add(wallet)
		}

		if !response.Pagination.HasNext {
			break
		}

		cursor = response.Pagination.NextCursor
	}

	return wallets, nil
}

func (ac apiCall) PrimeDescribeProducts() (ProductLookup, error) {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout())
	defer cancel()

	products := make(ProductLookup)

	var cursor string

	for {

		request := &prime.ListProductsRequest{
			PortfolioId: ac.portfolioId,
			Pagination:  &prime.PaginationParams{Cursor: cursor},
		}

		response, err := ac.config.PrimeClient.ListProducts(ctx, request)
		if err != nil {
			return nil, err
		}

		for _, p := range response.Products {
			products.Add(p)
		}

		if !response.Pagination.HasNext {
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

func (ac apiCall) PrimeDescribeTradingBalances() ([]*prime.Balance, error) {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout())
	defer cancel()

	response, err := ac.config.PrimeClient.ListWalletBalances(
		ctx,
		&prime.ListWalletBalancesRequest{
			PortfolioId: ac.portfolioId,
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

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout())
	defer cancel()

	round := amount.RoundFloor(ac.config.StablecoinFiatDigits)

	if round.IsZero() {
		return nil
	}

	zap.L().Info(
		"fiat conversion",
		zap.String("sourceSymbol", sourceWallet.Symbol),
		zap.String("destinationSymbol", destinationWallet.Symbol),
		zap.Any("amount", round),
	)

	request := &prime.CreateConversionRequest{
		PortfolioId:         ac.portfolioId,
		SourceWalletId:      sourceWallet.Id,
		DestinationWalletId: destinationWallet.Id,
		SourceSymbol:        strings.ToUpper(sourceWallet.Symbol),
		DestinationSymbol:   strings.ToUpper(destinationWallet.Symbol),
		Amount:              round.String(),
		IdempotencyKey:      uuid.New().String(),
	}

	response, err := ac.config.PrimeClient.CreateConversion(ctx, request)
	if err != nil {
		return err
	}

	zap.L().Info(
		"fiat conversion submitted",
		zap.String("sourceSymbol", sourceWallet.Symbol),
		zap.String("destinationSymbol", destinationWallet.Symbol),
		zap.Any("amount", round),
		zap.String("activityId", response.ActivityId),
	)

	return nil
}

func (ac apiCall) PrimeCreateMarketOrder(
	productId string,
	value,
	orderSize decimal.Decimal,
	asset *prime.Balance,
) error {
	holds, err := asset.HoldsNum()
	if err != nil {
		return err
	}

	clientOrderId := generateUniqueId(
		productId,
		prime.OrderSideSell,
		prime.OrderTypeMarket,
		prime.TimeInForceGoodUntilTime,
		orderSize.String(),
		holds.String(),
	)

	if _, exists := ac.ordersCache.Get(clientOrderId); exists == nil {
		return nil
	}

	zap.L().Info(
		"create market order request",
		zap.String("symbol", asset.Symbol),
		zap.Any("amount", asset.Amount),
		zap.Any("value", value),
		zap.Any("orderSize", orderSize),
	)

	request := ac.createMarketOrderRequest(
		productId,
		value,
		orderSize,
		asset,
		clientOrderId,
	)

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout())
	defer cancel()

	response, err := ac.config.PrimeClient.CreateOrder(ctx, request)
	if err != nil {
		return fmt.Errorf(
			"unable to create market order - client order id: %s - symbol: %s - size: %v %w",
			clientOrderId,
			asset.Symbol,
			orderSize,
			err,
		)
	}

	ac.ordersCache.Set(clientOrderId, response.OrderId)

	zap.L().Info(
		"market order created",
		zap.String("orderId", response.OrderId),
		zap.String("clientOrderId", clientOrderId),
	)

	return nil
}

func (ac apiCall) createMarketOrderRequest(
	productId string,
	value,
	orderSize decimal.Decimal,
	asset *prime.Balance,
	clientOrderId string,
) *prime.CreateOrderRequest {

	return &prime.CreateOrderRequest{
		Order: &prime.Order{
			PortfolioId:   ac.portfolioId,
			ProductId:     productId,
			Side:          prime.OrderSideSell,
			Type:          prime.OrderTypeMarket,
			ClientOrderId: clientOrderId,
			BaseQuantity:  orderSize.String(),
		},
	}
}

func (ac apiCall) PrimeCreateTwapOrder(
	productId string,
	value,
	orderSize,
	limitPrice decimal.Decimal,
	asset *prime.Balance,
) error {

	holds, err := asset.HoldsNum()
	if err != nil {
		return err
	}

	clientOrderId := generateUniqueId(
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

	zap.L().Info(
		"create twap order request",
		zap.String("symbol", asset.Symbol),
		zap.Any("amount", asset.Amount),
		zap.Any("value", value),
		zap.Any("orderSize", orderSize),
	)

	request := ac.createTwapOrderRequest(
		productId,
		value,
		orderSize,
		asset,
		limitPrice,
		ac.config.TwapDuration(),
		clientOrderId,
	)

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout())
	defer cancel()

	response, err := ac.config.PrimeClient.CreateOrder(ctx, request)
	if err != nil {
		return fmt.Errorf(
			"unable to create twap order - client order id: %s - symbol: %s - size: %v %w",
			clientOrderId,
			asset.Symbol,
			orderSize,
			err,
		)
	}

	ac.ordersCache.Set(clientOrderId, response.OrderId)

	zap.L().Info(
		"twap order created",
		zap.String("orderId", response.OrderId),
		zap.String("clientOrderId", clientOrderId),
	)

	return nil
}

func (ac apiCall) createTwapOrderRequest(
	productId string,
	value,
	orderSize decimal.Decimal,
	asset *prime.Balance,
	limitPrice decimal.Decimal,
	duration time.Duration,
	clientOrderId string,
) *prime.CreateOrderRequest {

	startTime := time.Now()
	endTime := startTime.Add(duration)

	return &prime.CreateOrderRequest{
		Order: &prime.Order{
			PortfolioId:   ac.portfolioId,
			ProductId:     productId,
			Side:          prime.OrderSideSell,
			Type:          prime.OrderTypeTwap,
			TimeInForce:   prime.TimeInForceGoodUntilTime,
			ClientOrderId: clientOrderId,
			BaseQuantity:  orderSize.String(),
			LimitPrice:    limitPrice.String(),
			StartTime:     startTime.Format("2006-01-02T15:04:05Z"),
			ExpiryTime:    endTime.Format("2006-01-02T15:04:05Z"),
		},
	}
}

func (ac apiCall) ExchangeCurrentProductPrice(productId string) (decimal.Decimal, error) {
	return exchange.CurrentProductPrice(productId, ac.config.PrimeCallTimeout(), ac.config.HttpClient)
}
