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

package monitor

import (
	"fmt"
	"strings"
	"time"

	"github.com/coinbase-samples/prime-liquidator-go/config"
	"github.com/coinbase-samples/prime-liquidator-go/monitor/caller"
	"github.com/coinbase-samples/prime-liquidator-go/util"
	prime "github.com/coinbase-samples/prime-sdk-go"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Liquidator struct {
	config         *config.AppConfig
	convertSymbols caller.ConvertSymbols
	balances       []*prime.Balance
	products       caller.ProductLookup
	wallets        caller.WalletLookup
	call           caller.Caller
}

// RunLiquidator continuously assets and changes them into
// fiat.
func RunLiquidator(config *config.AppConfig) {

	l := newLiquidator(config)

	l.monitor()
}

// newLiquidator returns a new Liquidator struct pointer.
func newLiquidator(config *config.AppConfig) (l *Liquidator) {

	l = &Liquidator{
		config:         config,
		convertSymbols: make(caller.ConvertSymbols),
		call:           caller.NewCaller(config),
	}

	for _, s := range config.ConvertSymbols() {
		l.convertSymbols.Add(s)
	}

	return
}

// monitor continuously loops through the current trading balances
// and processes the assets. New sell TWAP orders are created if the
// asset is tradeable. If the asset is a stablecoin, then a conversion
// request is created.
func (l *Liquidator) monitor() {

	for {

		if err := l.describeCurrentState(); err != nil {
			zap.L().Error("unable to describe current state", zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}

		for _, asset := range l.balances {
			if err := l.processAsset(asset); err != nil {
				zap.L().Error("unable to process assets", zap.Error(err))
			}
			time.Sleep(500 * time.Millisecond)
		}

		time.Sleep(5 * time.Second)
	}
}

// describeCurrentState lookups up the trading wallets, balances,
// and products and sets updates the state on the struct.
func (l *Liquidator) describeCurrentState() (err error) {

	l.wallets, err = l.call.PrimeDescribeTradingWallets()
	if err != nil {
		return
	}

	l.products, err = l.call.PrimeDescribeProducts()
	if err != nil {
		return
	}

	l.balances, err = l.call.PrimeDescribeTradingBalances()
	if err != nil {
		return
	}

	return
}

// processConversion looks up the stablecoin and fiat wallets and then
// submits a Prime conversion request.
func (l Liquidator) processConversion(
	amount decimal.Decimal,
	asset *prime.Balance,
) error {

	fiatWallet := l.wallets.Lookup(l.config.FiatCurrencySymbol)
	if fiatWallet == nil {
		return fmt.Errorf("fiat wallet not found: %s", l.config.FiatCurrencySymbol)
	}

	stablecoinWallet := l.wallets.Lookup(asset.Symbol)
	if stablecoinWallet == nil {
		return fmt.Errorf("stablecoin wallet not found: %s", asset.Symbol)
	}

	return l.call.PrimeCreateConversion(stablecoinWallet, fiatWallet, amount)
}

// processAsset takes an asset and either creates a sell order for fiat or
// issues a conversion request if the asset is a stablecoin
func (l Liquidator) processAsset(asset *prime.Balance) error {
	if util.IsFiat(asset.Symbol) {
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
	if l.convertSymbols.Is(asset.Symbol) {
		return l.processConversion(amount, asset)
	}

	productId := l.productId(asset)

	price, err := l.call.ExchangeCurrentProductPrice(productId)
	if err != nil {
		return fmt.Errorf("cannot get exchange price: %s - err: %w", productId, err)
	}

	product := l.products.Lookup(productId)
	if product == nil {
		return fmt.Errorf("Unknown product id: %s", productId)
	}

	holds, err := asset.HoldsNum()
	if err != nil {
		return err
	}

	orderSize, err := l.call.PrimeCalculateOrderSize(product, amount, holds)
	if err != nil {
		return err
	}

	if orderSize.IsZero() {
		return nil
	}

	value := price.Mul(orderSize)

	if value.IsZero() {
		return nil
	}

	quoteMin, err := product.QuoteMinSizeNum()
	if err != nil {
		return err
	}

	// Ensure that that the order value is is equal to or greater than the quote min size
	if value.Cmp(quoteMin) < 0 {
		return nil
	}

	// Check to see if the size of the order fits into the TWAP requirements
	if meetsTwapRequirements(value, l.config.TwapMinNotional(), l.config.TwapDuration()) {

		limitPrice, err := l.calculateTwapLimitPrice(product, price)
		if err != nil {
			return err
		}

		return l.call.PrimeCreateTwapOrder(
			productId,
			value,
			orderSize,
			limitPrice,
			asset,
		)
	}

	// Create a market order
	return l.call.PrimeCreateMarketOrder(
		productId,
		value,
		orderSize,
		asset,
	)

}

// calculateTwapLimitPrice looks at the product, current
// price, and max discount and returns the adjusted TWAP
// price X% the most recent Exchange lookup.
func (l Liquidator) calculateTwapLimitPrice(
	product *prime.Product,
	price decimal.Decimal,
) (limitPrice decimal.Decimal, err error) {

	var quoteIncrement decimal.Decimal
	if quoteIncrement, err = product.QuoteIncrementNum(); err != nil {
		return
	}

	maxDiscount := price.Mul(l.config.TwapMaxDiscountPercent)

	limitPrice = l.adjustTwapLimitPrice(price.Sub(maxDiscount), quoteIncrement)
	return
}

func (l Liquidator) adjustTwapLimitPrice(
	price,
	quoteIncrement decimal.Decimal,
) decimal.Decimal {
	quo, rem := price.QuoRem(quoteIncrement, 0)

	if rem.IsZero() {
		return price
	}

	return quo.Floor().Mul(quoteIncrement)
}

func (l Liquidator) productId(asset *prime.Balance) string {
	return fmt.Sprintf("%s-%s", strings.ToUpper(asset.Symbol), strings.ToUpper(l.config.FiatCurrencySymbol))
}
