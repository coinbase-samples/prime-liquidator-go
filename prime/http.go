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

package prime

import (
	"net/http"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func init() {
	initHttpTransPort()
}

var httpTransport *http.Transport

func SetHttpTransport(tr *http.Transport) *http.Transport {
	httpTransport = tr
	return httpTransport
}

func GetHttpTransport() *http.Transport {
	return httpTransport
}

func initHttpTransPort() *http.Transport {
	if httpTransport != nil {
		return httpTransport
	}

	maxIdleConnections := 50
	max := os.Getenv("PRIME_SDK_MAX_IDLE_CONNNECTIONS")
	if len(max) > 0 {
		n, err := strconv.ParseInt(max, 10, 0)
		if err != nil {
			log.Fatalf("unable to parse PRIME_SDK_MAX_IDLE_CONNNECTIONS %w", err)
		}
		maxIdleConnections = int(n)
	}

	httpTransport = &http.Transport{
		MaxIdleConns:       maxIdleConnections,
		DisableKeepAlives:  false,
		DisableCompression: false,
	}

	return httpTransport
}
