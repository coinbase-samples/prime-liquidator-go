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
	"time"

	"github.com/coinbase-samples/prime-liquidator-go/prime"
	log "github.com/sirupsen/logrus"
)

var toConvertSymbols = map[string]bool{
	"usdc": true,
}

type ProductLookup map[string]*prime.Product

type WalletLookup map[string]*prime.Wallet

// TODO: Make an env var
const fiatCurrencySymbol = "USD"

const twapDuration = time.Minute * 5

func ConvertToFiat() {

	portfolioId := prime.GetCredentials().PortfolioId

	for {

		products, wallets, balances, err := describeCurrentState(portfolioId)

		if err != nil {
			log.Error(err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, asset := range balances {
			if err := processAsset(portfolioId, asset, wallets, products); err != nil {
				log.Error(err)
			}
			time.Sleep(1 * time.Second)
		}

		time.Sleep(5 * time.Second)
	}
}