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

package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	prime "github.com/coinbase-samples/prime-sdk-go"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

var exchangeApiBaseUrl = "https://api.exchange.coinbase.com"

func init() {
	baseUrl := os.Getenv("COINBASE_EXCHANGE_BASE_URL")
	if len(baseUrl) > 0 {
		_, err := url.Parse(baseUrl)
		if err != nil {
			zap.L().Fatal(
				"cannot parse COINBASE_EXCHANGE_BASE_URL",
				zap.String("received", baseUrl),
				zap.Error(err),
			)
		}
		exchangeApiBaseUrl = baseUrl
	}
}

type ExchangeProductPrice struct {
	Price string `json:"price"`
}

func CurrentProductPrice(productId string, timeout time.Duration, httpClient *http.Client) (decimal.Decimal, error) {

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var price decimal.Decimal

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/products/%s/ticker", exchangeApiBaseUrl, productId),
		nil,
	)
	if err != nil {
		return price, fmt.Errorf("cannot create Exchange product price request: %w", err)
	}

	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return price, fmt.Errorf("cannot call Exchange product err: %w", err)
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return price, fmt.Errorf("cannot read Exchange product price response - err: %w", err)
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
