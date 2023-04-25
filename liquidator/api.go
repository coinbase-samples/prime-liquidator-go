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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/coinbase-samples/prime-liquidator-go/prime"
	"github.com/google/uuid"
	"github.com/jellydator/ttlcache/v2"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

type ExchangeProductPrice struct {
	Price string `json:"price"`
}

type ApiCall struct {
	config      AppConfig
	ordersCache *ttlcache.Cache
}

func newApiCall(config AppConfig) (a ApiCall) {

	a = ApiCall{config: config}

	a.ordersCache = ttlcache.NewCache()
	a.ordersCache.SetTTL(config.TwapDuration)
	a.ordersCache.SetCacheSizeLimit(config.OrdersCacheSize)
	return
}

func (ac ApiCall) describeTradingWallets() (WalletLookup, error) {

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
			wallets.add(wallet)
		}

		if !response.HasNext() {
			break
		}

		cursor = response.Pagination.NextCursor
	}

	return wallets, nil
}

func (ac ApiCall) describeProducts() (ProductLookup, error) {

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
			products.add(p)
		}

		if !response.HasNext() {
			break
		}

		cursor = response.Pagination.NextCursor
	}

	return products, nil
}

func (ac ApiCall) calculateOrderSize(
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

func (ac ApiCall) describeTradingBalances() ([]*prime.AssetBalances, error) {

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

func (ac ApiCall) createConversion(
	sourceWallet,
	destinationWallet *prime.Wallet,
	amount decimal.Decimal,
) error {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout)
	defer cancel()

	round := amount.RoundFloor(2)

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

func (ac ApiCall) createOrder(
	productId string,
	value,
	orderSize decimal.Decimal,
	asset *prime.AssetBalances,
	limitPrice decimal.Decimal,
	duration time.Duration,
) error {

	clientOrderId := prime.GenerateUniqueId(
		productId,
		prime.OrderSideSell,
		prime.OrderTypeTwap,
		prime.TimeInForceGoodUntilTime,
		orderSize.String(),
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
		duration,
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

func (ac ApiCall) createOrderRequest(
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

func (ac ApiCall) currentExchangeProductPrice(productId string) (decimal.Decimal, error) {

	ctx, cancel := context.WithTimeout(context.Background(), ac.config.PrimeCallTimeout)
	defer cancel()

	var price decimal.Decimal

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("https://api.exchange.coinbase.com/products/%s/ticker", productId),
		nil,
	)
	if err != nil {
		return price, fmt.Errorf("unable to create Exchange product price request: %w", err)
	}

	req.Header.Add("Accept", "application/json")

	client := http.Client{Transport: prime.GetHttpTransport()}

	res, err := client.Do(req)
	if err != nil {
		return price, fmt.Errorf("cannot call Exchange product err: %w", err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return price, fmt.Errorf("unable to read Exchange product price response - err: %w", err)
	}

	if res.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "message") {
		var errMsg prime.ErrorMessage
		if err := json.Unmarshal(body, &errMsg); err != nil {
			return price, fmt.Errorf("cannot unmarshal Exchange error messsage: %w", err)
		}

		return price, fmt.Errorf("cannot fetch Exchnage price price - did return 200 - val: %d - msg: %s", res.StatusCode, errMsg.Value)
	}

	if res.StatusCode != http.StatusOK {
		return price, fmt.Errorf("exchange product price did return 200 - val: %d", res.StatusCode)
	}

	var productPrice ExchangeProductPrice
	if err = json.Unmarshal(body, &productPrice); err != nil {
		return price, fmt.Errorf(
			"cannot parse Exchange product price response - value: %s - err: %w",
			string(body),
			err,
		)
	}

	v, err := decimal.NewFromString(productPrice.Price)
	if err != nil {
		return price, fmt.Errorf(
			"unable to parse Exchange product price - value: %s - err: %w",
			productPrice.Price,
			err,
		)
	}

	price = v

	return price, nil
}
