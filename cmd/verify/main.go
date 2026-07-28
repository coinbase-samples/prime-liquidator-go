/**
 * Copyright 2026-present Coinbase Global, Inc.
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/coinbase-samples/prime-liquidator-go/config"
	"github.com/coinbase-samples/prime-liquidator-go/exchange"
	"github.com/coinbase/prime-sdk-go/balances"
	"github.com/coinbase/prime-sdk-go/client"
	"github.com/coinbase/prime-sdk-go/credentials"
	"github.com/coinbase/prime-sdk-go/model"
	"github.com/coinbase/prime-sdk-go/portfolios"
	"github.com/coinbase/prime-sdk-go/wallets"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "verify-setup failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	missing := missingEnv()
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment: %s", strings.Join(missing, ", "))
	}

	creds, err := credentials.ReadEnvCredentials("PRIME_CREDENTIALS")
	if err != nil {
		return fmt.Errorf("cannot read PRIME_CREDENTIALS: %w", err)
	}

	if err := validateCredentialsShape(creds); err != nil {
		return err
	}

	httpClient, err := config.InitHttpClient(&config.AppConfig{
		HttpConnectTimeoutInSeconds: envOrDefault("HTTP_CONNECT_TIMEOUT", "5"),
		HttpConnKeepAliveInSeconds:  envOrDefault("HTTP_CONN_KEEP_ALIVE", "30"),
		HttpExpectContinueInSeconds: envOrDefault("HTTP_EXPECT_CONTINUE", "1"),
		HttpIdleConnInSeconds:       envOrDefault("HTTP_IDLE_CONN", "90"),
		HttpMaxAllIdleConnsCount:    envOrDefault("HTTP_MAX_ALL_IDLE_CONNS", "10"),
		HttpMaxHostIdleConnsCount:   envOrDefault("HTTP_MAX_HOST_IDLE_CONNS", "5"),
		HttpResponseHeaderInSeconds: envOrDefault("HTTP_RESPONSE_HEADER", "5"),
		HttpTLSHandshakeInSeconds:   envOrDefault("HTTP_TLS_HANDSHAKE", "5"),
	})
	if err != nil {
		return fmt.Errorf("cannot init http client: %w", err)
	}

	restClient := client.NewRestClient(creds, *httpClient)
	timeout := primeCallTimeout()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	portfoliosSvc := portfolios.NewPortfoliosService(restClient)
	portfolioResp, err := portfoliosSvc.ListPortfolios(ctx, &portfolios.ListPortfoliosRequest{})
	if err != nil {
		return fmt.Errorf("prime API call failed (list portfolios): %w", err)
	}

	foundPortfolio := false
	for _, p := range portfolioResp.Portfolios {
		if p.Id == creds.PortfolioId {
			foundPortfolio = true
			fmt.Printf("Prime API access OK (portfolio %s)\n", p.Name)
			break
		}
	}
	if !foundPortfolio {
		return fmt.Errorf("portfolio id %s not found in ListPortfolios response", creds.PortfolioId)
	}

	walletsSvc := wallets.NewWalletsService(restClient)
	walletResp, err := walletsSvc.ListWallets(ctx, &wallets.ListWalletsRequest{
		PortfolioId: creds.PortfolioId,
		Type:        model.WalletTypeTrading,
	})
	if err != nil {
		return fmt.Errorf("prime API call failed (list trading wallets): %w", err)
	}
	fmt.Printf("Trading wallets: %d\n", len(walletResp.Wallets))

	balancesSvc := balances.NewBalancesService(restClient)
	balanceResp, err := balancesSvc.ListPortfolioBalances(ctx, &balances.ListPortfolioBalancesRequest{
		PortfolioId: creds.PortfolioId,
		Type:        model.BalanceTypeTrading,
	})
	if err != nil {
		return fmt.Errorf("prime API call failed (list trading balances): %w", err)
	}

	nonZero := 0
	for _, b := range balanceResp.Balances {
		amount, err := b.AmountNum()
		if err != nil {
			return fmt.Errorf("invalid balance amount for %s: %w", b.Symbol, err)
		}
		if amount.IsZero() {
			continue
		}
		nonZero++
		holds, err := b.HoldsNum()
		if err != nil {
			return fmt.Errorf("invalid balance holds for %s: %w", b.Symbol, err)
		}
		fmt.Printf("  %s amount=%s holds=%s\n", strings.ToUpper(b.Symbol), formatDecimal(amount), formatDecimal(holds))
	}
	fmt.Printf("Non-zero trading balances: %d\n", nonZero)

	fiat := envOrDefault("FIAT_CURRENCY_SYMBOL", "USD")
	productId := fmt.Sprintf("BTC-%s", strings.ToUpper(fiat))
	price, err := exchange.CurrentProductPrice(productId, timeout, httpClient)
	if err != nil {
		return fmt.Errorf("coinbase exchange price check failed for %s: %w", productId, err)
	}
	fmt.Printf("Exchange price check OK (%s = %s)\n", productId, price.String())

	dryRun := strings.EqualFold(envOrDefault("DRY_RUN", "true"), "true")
	if dryRun {
		fmt.Println("DRY_RUN=true (default): liquidator will not submit orders or conversions")
	} else {
		fmt.Println("WARNING: DRY_RUN=false — running the liquidator will place real orders and conversions")
	}

	fmt.Println("verify-setup completed successfully")
	return nil
}

func missingEnv() []string {
	var missing []string
	if os.Getenv("PRIME_CREDENTIALS") == "" {
		missing = append(missing, "PRIME_CREDENTIALS")
	}
	return missing
}

func validateCredentialsShape(creds *credentials.Credentials) error {
	var missing []string
	if creds.AccessKey == "" {
		missing = append(missing, "accessKey")
	}
	if creds.Passphrase == "" {
		missing = append(missing, "passphrase")
	}
	if creds.SigningKey == "" {
		missing = append(missing, "signingKey")
	}
	if creds.PortfolioId == "" {
		missing = append(missing, "portfolioId")
	}
	if len(missing) > 0 {
		return fmt.Errorf("PRIME_CREDENTIALS JSON is missing fields: %s", strings.Join(missing, ", "))
	}

	// Ensure JSON parses without logging secrets.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal([]byte(os.Getenv("PRIME_CREDENTIALS")), &probe); err != nil {
		return fmt.Errorf("PRIME_CREDENTIALS is not valid JSON: %w", err)
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func primeCallTimeout() time.Duration {
	app := &config.AppConfig{PrimeCallTimeoutInSeconds: envOrDefault("PRIME_CALL_TIMEOUT", "10")}
	return app.PrimeCallTimeout()
}

func formatDecimal(d decimal.Decimal) string {
	return d.StringFixedBank(8)
}
