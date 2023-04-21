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
			if err := processBalance(portfolioId, asset, wallets, products); err != nil {
				log.Error(err)
			}
			time.Sleep(1 * time.Second)
		}

		time.Sleep(5 * time.Second)
	}
}
