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
