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
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	euroSymbol = "eur"
	usdSymbol  = "usd"
)

func meetsTwapRequirements(
	value decimal.Decimal,
	twapMinNotional int,
	twapDuration time.Duration,
) bool {

	minNotionalPerHour := decimal.NewFromInt(int64(twapMinNotional))

	hours := decimal.NewFromFloat(twapDuration.Hours())

	return value.Div(hours).GreaterThanOrEqual(minNotionalPerHour)
}

func isFiat(symbol string) (f bool) {
	v := strings.ToLower(symbol)
	if v == usdSymbol {
		f = true
	} else if v == euroSymbol {
		f = true
	}
	return
}
