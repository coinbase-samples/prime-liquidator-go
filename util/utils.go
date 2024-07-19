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

package util

import (
	"crypto/md5"
	"fmt"
	"strings"
)

func GenerateUniqueId(params ...string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(params, "-"))))
}

func IsFiat(symbol string) (f bool) {
	if strings.ToLower(symbol) == "usd" {
		f = true
	}
	return
}