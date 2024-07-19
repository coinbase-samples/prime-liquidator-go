/**
 * Copyright 2024-present Coinbase Global, Inc.
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

package config

import (
	"net"
	"net/http"

	"golang.org/x/net/http2"
)

func InitHttpClient(appConfig *AppConfig) (*http.Client, error) {

	tr := &http.Transport{
		ResponseHeaderTimeout: appConfig.HttpResponseHeader(),
		Proxy:                 http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			KeepAlive: appConfig.HttpConnKeepAlive(),
			DualStack: true,
			Timeout:   appConfig.HttpConnectTimeout(),
		}).DialContext,
		MaxIdleConns:          appConfig.HttpMaxAllIdleConns(),
		IdleConnTimeout:       appConfig.HttpIdleConn(),
		TLSHandshakeTimeout:   appConfig.HttpTLSHandshake(),
		MaxIdleConnsPerHost:   appConfig.HttpMaxHostIdleConns(),
		ExpectContinueTimeout: appConfig.HttpExpectContinue(),
	}

	if err := http2.ConfigureTransport(tr); err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: tr,
	}, nil
}
