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

package caller

import (
	"strings"

	"github.com/coinbase-samples/prime-liquidator-go/prime"
)

type ProductLookup map[string]*prime.Product

func (pl ProductLookup) Lookup(id string) *prime.Product {
	if p, found := pl[id]; !found {
		return nil
	} else {
		return p
	}
}

func (pl ProductLookup) Add(p *prime.Product) {
	pl[p.Id] = p
}

type WalletLookup map[string]*prime.Wallet

func (wl WalletLookup) Lookup(id string) *prime.Wallet {
	if w, found := wl[strings.ToUpper(id)]; !found {
		return nil
	} else {
		return w
	}
}

func (wl WalletLookup) Add(w *prime.Wallet) {
	wl[w.Symbol] = w
}

type ConvertSymbols map[string]bool

func (cs ConvertSymbols) Is(v string) (c bool) {
	if _, found := cs[v]; found {
		c = true
	}
	return
}

func (cs ConvertSymbols) Add(v string) {
	cs[v] = true
}
