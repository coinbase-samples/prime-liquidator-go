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
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

const primeCallTimeoutInSeconds = 30 * time.Second

func describeCurrentState(portfolioId string) (
	products ProductLookup,
	wallets WalletLookup,
	balances []*prime.AssetBalances,
	err error,
) {

	wallets, err = describeTradingWallets(portfolioId)
	if err != nil {
		return
	}

	products, err = describeProducts(portfolioId)
	if err != nil {
		return
	}

	balances, err = describeTradingBalances(portfolioId)
	if err != nil {
		return
	}

	return
}

func createOrder(
	portfolioId,
	productId string,
	value,
	orderSize decimal.Decimal,
	asset *prime.AssetBalances,
	limitPrice decimal.Decimal,
	duration time.Duration,
	clientOrderId string,
) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), primeCallTimeoutInSeconds)
	defer cancel()

	log.Infof("create order request - asset: %s - balance: %s - value: %v - order size: %v", asset.Symbol, asset.Amount, value, orderSize)

	request := createOrderRequest(portfolioId,
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
			"unable to create order - client order id: %s - err: %w",
			clientOrderId,
			err,
		)
	}

	log.Infof("new order created - id: %s", response.OrderId)

	return response.OrderId, nil
}

func createOrderRequest(
	portfolioId,
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
		PortfolioId:   portfolioId,
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

func createOrderPreview(
	portfolioId,
	productId string,
	value,
	orderSize decimal.Decimal,
	asset *prime.AssetBalances,
	limitPrice decimal.Decimal,
	duration time.Duration,
	clientOrderId string,
) (*prime.CreateOrderPreviewResponse, error) {

	ctx, cancel := context.WithTimeout(context.Background(), primeCallTimeoutInSeconds)
	defer cancel()

	request := createOrderRequest(portfolioId,
		productId,
		value,
		orderSize,
		asset,
		limitPrice,
		duration,
		clientOrderId,
	)

	response, err := prime.CreateOrderPreview(ctx, request)
	if err != nil {
		return nil, err
	}

	return response, nil

}

func describeTradingBalances(portfolioId string) ([]*prime.AssetBalances, error) {

	ctx, cancel := context.WithTimeout(context.Background(), primeCallTimeoutInSeconds)
	defer cancel()

	response, err := prime.DescribeBalances(
		ctx,
		&prime.DescribeBalancesRequest{PortfolioId: portfolioId},
	)

	if err != nil {
		return nil, err
	}

	return response.Balances, nil
}

func describeProducts(portfolioId string) (ProductLookup, error) {

	ctx, cancel := context.WithTimeout(context.Background(), primeCallTimeoutInSeconds)
	defer cancel()

	products := make(map[string]*prime.Product)

	var cursor string

	for {

		request := &prime.DescribeProductsRequest{
			PortfolioId:    portfolioId,
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

func describeTradingWallets(portfolioId string) (WalletLookup, error) {

	ctx, cancel := context.WithTimeout(context.Background(), primeCallTimeoutInSeconds)
	defer cancel()

	var cursor string

	wallets := make(map[string]*prime.Wallet)

	for {

		request := &prime.DescribeWalletsRequest{
			PortfolioId: portfolioId,
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

		if response.HasNext() {
			break
		}

		cursor = response.Pagination.NextCursor
	}

	return wallets, nil
}

func createConversion(
	portfolioId string,
	sourceWallet,
	destinationWallet *prime.Wallet,
	amount decimal.Decimal,
) error {

	ctx, cancel := context.WithTimeout(context.Background(), primeCallTimeoutInSeconds)
	defer cancel()

	round := amount.RoundFloor(2)

	if round.IsZero() {
		return nil
	}

	log.Infof("converting %s to %s - amount: %v", sourceWallet.Symbol, destinationWallet.Symbol, round)

	request := &prime.CreateConversionRequest{
		PortfolioId:         portfolioId,
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
