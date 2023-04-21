package liquidator

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/coinbase-samples/prime-liquidator-go/prime"
	"github.com/shopspring/decimal"
)

type ExchangeProductPrice struct {
	Price string `json:"price"`
}

func currentExchangeProductPrice(productId string) (decimal.Decimal, error) {

	ctx, cancel := context.WithTimeout(context.Background(), primeCallTimeoutInSeconds)
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
		return price, fmt.Errorf("unable call Exchange product err: %v", err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return price, fmt.Errorf("unable to read Exchange product price response - err: %w", err)
	}

	if res.StatusCode == http.StatusBadRequest && strings.Contains(string(body), "message") {
		var errMsg prime.ErrorMessage
		if err := json.Unmarshal(body, &errMsg); err != nil {
			return price, fmt.Errorf("unable to unmarshal Exchange error messsage: %v", err)
		}

		return price, fmt.Errorf("unable to fetch Exchnage price price - did return 200 - val: %d - msg: %s", res.StatusCode, errMsg.Message)
	}

	if res.StatusCode != http.StatusOK {
		return price, fmt.Errorf("exchange product price did return 200 - val: %d", res.StatusCode)
	}

	var productPrice ExchangeProductPrice
	if err = json.Unmarshal(body, &productPrice); err != nil {
		return price, fmt.Errorf(
			"unable to parse Exchange product price response - value: %s - err: %w",
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
