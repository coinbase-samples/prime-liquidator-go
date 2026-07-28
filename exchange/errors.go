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

package exchange

import (
	"errors"
	"fmt"
	"strings"
)

// PriceUnavailableError is returned when Coinbase Exchange cannot quote a product
// (for example delisted or unknown products). Callers may skip liquidating the asset.
type PriceUnavailableError struct {
	ProductID  string
	StatusCode int
	Message    string
}

func (e *PriceUnavailableError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("exchange price unavailable for %s: %s", e.ProductID, e.Message)
	}
	return fmt.Sprintf("exchange price unavailable for %s (HTTP %d)", e.ProductID, e.StatusCode)
}

// IsPriceUnavailable reports whether err indicates the product cannot be priced on Exchange.
func IsPriceUnavailable(err error) bool {
	var target *PriceUnavailableError
	return errors.As(err, &target)
}

func isUnavailableExchangeResponse(statusCode int, message string) bool {
	if statusCode == 404 {
		return true
	}
	if statusCode != 400 {
		return false
	}
	lower := strings.ToLower(message)
	return strings.Contains(lower, "delisted") ||
		strings.Contains(lower, "not found") ||
		strings.Contains(lower, "not allowed")
}
