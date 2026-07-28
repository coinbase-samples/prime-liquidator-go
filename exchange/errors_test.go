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

import "testing"

func TestIsPriceUnavailable(t *testing.T) {
	err := &PriceUnavailableError{
		ProductID:  "MATIC-USD",
		StatusCode: 400,
		Message:    "Not allowed for delisted products",
	}
	if !IsPriceUnavailable(err) {
		t.Fatal("expected delisted price error to be unavailable")
	}
	if IsPriceUnavailable(nil) {
		t.Fatal("nil should not be unavailable")
	}
}

func TestIsUnavailableExchangeResponse(t *testing.T) {
	cases := []struct {
		code    int
		message string
		want    bool
	}{
		{400, "Not allowed for delisted products", true},
		{404, "", true},
		{400, "Product not found", true},
		{500, "internal error", false},
		{429, "rate limited", false},
	}
	for _, tc := range cases {
		if got := isUnavailableExchangeResponse(tc.code, tc.message); got != tc.want {
			t.Fatalf("code=%d message=%q: got %v want %v", tc.code, tc.message, got, tc.want)
		}
	}
}
