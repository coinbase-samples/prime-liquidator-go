package liquidator

import (
	"fmt"
	"strings"
	"time"

	"github.cbhq.net/ryan-nitz/prime-liquidator/prime"
	"github.com/jellydator/ttlcache/v2"
)

var ordersCache *ttlcache.Cache

func init() {
	ordersCache = ttlcache.NewCache()
	ordersCache.SetTTL(time.Duration(10 * time.Minute))
	ordersCache.SetCacheSizeLimit(1000)
}

func processBalance(
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
		return fmt.Errorf("Unable to get exchange price: %s - err: %v", asset.Symbol, err)
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
