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
	prime "github.com/coinbase-samples/prime-sdk-go"
	"github.com/shopspring/decimal"
)

type Caller interface {
	ExchangeCurrentProductPrice(productId string) (decimal.Decimal, error)
	PrimeDescribeTradingWallets() (WalletLookup, error)
	PrimeDescribeProducts() (ProductLookup, error)
	PrimeDescribeTradingBalances() ([]*prime.Balance, error)
	PrimeCreateConversion(sourceWallet, destinationWallet *prime.Wallet, amount decimal.Decimal) error

	PrimeCreateTwapOrder(
		productId string,
		value,
		orderSize,
		limitPrice decimal.Decimal,
		asset *prime.Balance,
	) error

	PrimeCreateMarketOrder(
		productId string,
		value,
		orderSize decimal.Decimal,
		asset *prime.Balance,
	) error

	PrimeCalculateOrderSize(product *prime.Product, amount, holds decimal.Decimal) (orderSize decimal.Decimal, err error)
}
