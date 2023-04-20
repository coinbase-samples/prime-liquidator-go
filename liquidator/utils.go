package liquidator

import (
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

const twapMaxDiscountPercent = 0.1

func calculateTwapLimitPrice(
	productId string,
	price decimal.Decimal,
	products ProductLookup,
) (limitPrice decimal.Decimal, err error) {

	var (
		quoteIncrement decimal.Decimal
	)

	product, found := products[productId]
	if !found {
		err = fmt.Errorf("Unknown product id: %s", productId)
		return
	}

	if quoteIncrement, err = product.QuoteIncrementNum(); err != nil {
		return
	}

	maxDiscount := price.Mul(decimal.NewFromFloat32(twapMaxDiscountPercent))

	limitPrice = adjustTwapLimitPrice(price.Sub(maxDiscount), quoteIncrement)
	return

}

func adjustTwapLimitPrice(price, quoteIncrement decimal.Decimal) decimal.Decimal {
	quo, rem := price.QuoRem(quoteIncrement, 0)

	if rem.IsZero() {
		return price
	}

	return quo.Floor().Mul(quoteIncrement)
}

func calculateOrderSize(
	productId string,
	amount decimal.Decimal,
	holds decimal.Decimal,
	products ProductLookup,
) (orderSize decimal.Decimal, err error) {

	var (
		baseMin       decimal.Decimal
		baseMax       decimal.Decimal
		baseIncrement decimal.Decimal
	)

	product, found := products[productId]
	if !found {
		err = fmt.Errorf("Unknown product id: %s", productId)
		return
	}

	if baseMin, err = product.BaseMinSizeNum(); err != nil {
		return
	}

	if baseMax, err = product.BaseMaxSizeNum(); err != nil {
		return
	}

	if baseIncrement, err = product.BaseIncrementNum(); err != nil {
		return
	}

	availableAmount := amount.Sub(holds)

	if availableAmount.IsZero() {
		orderSize = availableAmount
		return
	}

	if availableAmount.IsNegative() {

		orderSize = decimal.NewFromInt(0)

	} else {

		orderSize = adjustOrderSize(availableAmount, baseMin, baseMax, baseIncrement)
	}

	return
}

func adjustOrderSize(amount, baseMin, baseMax, baseIncrement decimal.Decimal) decimal.Decimal {

	if amount.Cmp(baseMax) > 0 {
		return baseMax
	}

	if amount.Cmp(baseMin) < 0 {
		return decimal.NewFromFloat(0)
	}

	quo, rem := amount.QuoRem(baseIncrement, 0)

	if rem.IsZero() {
		return amount
	}

	return quo.Floor().Mul(baseIncrement)
}

func generateUniqueId(params ...string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(params, "-"))))
}
