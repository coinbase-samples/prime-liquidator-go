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
)

var ordersCache *ttlcache.Cache

func init() {
	ordersCache = ttlcache.NewCache()
	ordersCache.SetTTL(time.Duration(10 * time.Minute))
	ordersCache.SetCacheSizeLimit(1000)
}

func processAsset(
	portfolioId string,
	asset *prime.AssetBalances,
	wallets WalletLookup,
	products ProductLookup,
) error {

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
	if _, found := toConvertSymbols[asset.Symbol]; found {

		fiatWallet, found := wallets[fiatCurrencySymbol]

		if !found {
			return fmt.Errorf("fiat wallet not found: %s", fiatCurrencySymbol)
		}

		stablecoinWallet, found := wallets[strings.ToUpper(asset.Symbol)]

		if !found {
			return fmt.Errorf("stablecoin wallet not found: %s", asset.Symbol)
		}

		if err := createConversion(portfolioId, stablecoinWallet, fiatWallet, amount); err != nil {
			return err
		}

		return nil
	}

	price, err := currentExchangeProductPrice(
		strings.ToUpper(
			fmt.Sprintf("%s-%s", asset.Symbol, fiatCurrencySymbol),
		),
	)

	if err != nil {
		return fmt.Errorf("cannot get exchange price: %s - err: %w", asset.Symbol, err)
	}

	productId := fmt.Sprintf("%s-%s", strings.ToUpper(asset.Symbol), fiatCurrencySymbol)

	value := amount.Mul(price)

	if value.IsZero() {
		return nil
	}

	holds, err := asset.HoldsNum()
	if err != nil {
		return err
	}

	orderSize, err := calculateOrderSize(productId, amount, holds, products)
	if err != nil {
		return err
	}

	if orderSize.IsZero() {
		return nil
	}

	limitPrice, err := calculateTwapLimitPrice(productId, price, products)
	if err != nil {
		return err
	}

	clientOrderId := generateUniqueId(
		productId,
		prime.OrderSideSell,
		prime.OrderTypeTwap,
		prime.TimeInForceGoodUntilTime,
		orderSize.String(),
	)

	if _, exists := ordersCache.Get(clientOrderId); exists == nil {
		return nil
	}

	if orderId, err := createOrder(
		portfolioId,
		productId,
		value,
		orderSize,
		asset,
		limitPrice,
		twapDuration,
		clientOrderId,
	); err != nil {
		return err
	} else {
		ordersCache.Set(clientOrderId, orderId)
	}

	return nil
}