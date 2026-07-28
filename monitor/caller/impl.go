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
	"github.com/coinbase/prime-sdk-go/balances"
	"github.com/coinbase/prime-sdk-go/model"
	"github.com/coinbase/prime-sdk-go/orders"
	"github.com/coinbase/prime-sdk-go/products"
	"github.com/coinbase/prime-sdk-go/transactions"
	"github.com/coinbase/prime-sdk-go/utils"
	"github.com/coinbase/prime-sdk-go/wallets"
	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v2"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
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
		portfolioId: config.PrimeClient.Credentials().PortfolioId,
	}
}

func (ac apiCall) PrimeDescribeTradingWallets() (WalletLookup, error) {

	var cursor string

	walletLookup := make(WalletLookup)

	for {

		w, nextCursor, err := ac.primeListTradingWallets(cursor)

		if err != nil {
			return walletLookup, err
		}

		for _, wallet := range w {
			walletLookup.Add(wallet)
		}

		if len(nextCursor) == 0 {
			break
		}

		cursor = nextCursor
	}

	return walletLookup, nil
}

func (ac apiCall) primeListTradingWallets(cursor string) ([]*model.Wallet, string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout())
	defer cancel()

	request := &wallets.ListWalletsRequest{
		PortfolioId: ac.portfolioId,
		Type:        model.WalletTypeTrading,
		Pagination: &model.PaginationParams{
			Cursor: cursor,
		},
	}

	response, err := ac.config.Wallets.ListWallets(ctx, request)
	if err != nil {
		return nil, "", err
	}

	return response.Wallets, response.GetNextCursor(), nil
}

func (ac apiCall) PrimeDescribeProducts() (ProductLookup, error) {

	productLookup := make(ProductLookup)

	var cursor string

	for {

		p, nextCursor, err := ac.primeListProducts(cursor)

		if err != nil {
			return productLookup, err
		}

		for _, product := range p {
			productLookup.Add(product)
		}

		if len(nextCursor) == 0 {
			break
		}

		cursor = nextCursor
	}

	return productLookup, nil
}

func (ac apiCall) primeListProducts(cursor string) ([]*model.Product, string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout())
	defer cancel()

	request := &products.ListProductsRequest{
		PortfolioId: ac.portfolioId,
		Pagination:  &model.PaginationParams{Cursor: cursor},
	}

	response, err := ac.config.Products.ListProducts(ctx, request)
	if err != nil {
		return nil, "", err
	}

	return response.Products, response.GetNextCursor(), nil
}

func (ac apiCall) PrimeCalculateOrderSize(
	product *model.Product,
	amount,
	holds decimal.Decimal,
) (orderSize decimal.Decimal, err error) {
	orderSize, err = utils.CalculateOrderSize(product, amount, holds)
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

func (ac apiCall) PrimeDescribeTradingBalances() ([]*model.Balance, error) {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout())
	defer cancel()

	response, err := ac.config.Balances.ListPortfolioBalances(
		ctx,
		&balances.ListPortfolioBalancesRequest{
			PortfolioId: ac.portfolioId,
			Type:        model.BalanceTypeTrading,
		},
	)

	if err != nil {
		return nil, err
	}

	return response.Balances, nil
}

func (ac apiCall) PrimeCreateConversion(
	sourceWallet,
	destinationWallet *model.Wallet,
	amount decimal.Decimal,
) error {

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

	if ac.config.IsDryRun() {
		zap.L().Info(
			"dry run: skipping fiat conversion submission",
			zap.String("sourceSymbol", sourceWallet.Symbol),
			zap.String("destinationSymbol", destinationWallet.Symbol),
		)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout())
	defer cancel()

	request := &transactions.CreateConversionRequest{
		PortfolioId:         ac.portfolioId,
		SourceWalletId:      sourceWallet.Id,
		DestinationWalletId: destinationWallet.Id,
		SourceSymbol:        strings.ToUpper(sourceWallet.Symbol),
		DestinationSymbol:   strings.ToUpper(destinationWallet.Symbol),
		Amount:              round.String(),
		IdempotencyKey:      uuid.New().String(),
	}

	response, err := ac.config.Transactions.CreateConversion(ctx, request)
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
	asset *model.Balance,
) error {
	holds, err := asset.HoldsNum()
	if err != nil {
		return err
	}

	clientOrderId := generateUniqueId(
		productId,
		string(model.OrderSideSell),
		model.OrderTypeMarket,
		model.TimeInForceGoodUntilTime,
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

	if ac.config.IsDryRun() {
		zap.L().Info(
			"dry run: skipping market order submission",
			zap.String("symbol", asset.Symbol),
			zap.String("productId", productId),
		)
		return nil
	}

	request := ac.createMarketOrderRequest(
		productId,
		value,
		orderSize,
		asset,
		clientOrderId,
	)

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout())
	defer cancel()

	response, err := ac.config.Orders.CreateOrder(ctx, request)
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
	asset *model.Balance,
	clientOrderId string,
) *orders.CreateOrderRequest {

	return &orders.CreateOrderRequest{
		Order: &model.Order{
			PortfolioId:   ac.portfolioId,
			ProductId:     productId,
			Side:          string(model.OrderSideSell),
			Type:          model.OrderTypeMarket,
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
	asset *model.Balance,
) error {

	holds, err := asset.HoldsNum()
	if err != nil {
		return err
	}

	clientOrderId := generateUniqueId(
		productId,
		string(model.OrderSideSell),
		model.OrderTypeTwap,
		model.TimeInForceGoodUntilTime,
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

	if ac.config.IsDryRun() {
		zap.L().Info(
			"dry run: skipping twap order submission",
			zap.String("symbol", asset.Symbol),
			zap.String("productId", productId),
		)
		return nil
	}

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

	response, err := ac.config.Orders.CreateOrder(ctx, request)
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
	asset *model.Balance,
	limitPrice decimal.Decimal,
	duration time.Duration,
	clientOrderId string,
) *orders.CreateOrderRequest {

	startTime := time.Now()
	endTime := startTime.Add(duration)

	return &orders.CreateOrderRequest{
		Order: &model.Order{
			PortfolioId:   ac.portfolioId,
			ProductId:     productId,
			Side:          string(model.OrderSideSell),
			Type:          model.OrderTypeTwap,
			TimeInForce:   model.TimeInForceGoodUntilTime,
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
