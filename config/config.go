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

package config

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	prime "github.com/coinbase-samples/prime-sdk-go"
	"github.com/shopspring/decimal"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type AppConfig struct {
	PrimeClient                 *prime.Client
	HttpClient                  *http.Client
	HttpConnectTimeoutInSeconds string `mapstructure:"HTTP_CONNECT_TIMEOUT"`
	HttpConnKeepAliveInSeconds  string `mapstructure:"HTTP_CONN_KEEP_ALIVE"`
	HttpExpectContinueInSeconds string `mapstructure:"HTTP_EXPECT_CONTINUE"`
	HttpIdleConnInSeconds       string `mapstructure:"HTTP_IDLE_CONN"`
	HttpMaxAllIdleConnsCount    string `mapstructure:"HTTP_MAX_ALL_IDLE_CONNS"`
	HttpMaxHostIdleConnsCount   string `mapstructure:"HTTP_MAX_HOST_IDLE_CONNS"`
	HttpResponseHeaderInSeconds string `mapstructure:"HTTP_RESPONSE_HEADER"`
	HttpTLSHandshakeInSeconds   string `mapstructure:"HTTP_TLS_HANDSHAKE"`
	EnvName                     string `mapstructure:"ENV_NAME"`
	FiatCurrencySymbol          string `mapstructure:"FIAT_CURRENCY_SYMBOL"`
	TwapDurationInMinutes       string `mapstructure:"TWAP_DURATION"` // This should be at least 60'
	PrimeCallTimeoutInSeconds   string `mapstructure:"PRIME_CALL_TIMEOUT"`
	OrdersCacheSizeInItems      string `mapstructure:"ORDERS_CACHE_SIZE"`
	ConvertSymbolsArray         string `mapstructure:"CONVERT_SYMBOLS"`
	TwapMinNotionalPerHour      string `mapstructure:"TWAP_MIN_NOTIONAL"`

	TwapMaxDiscountPercent decimal.Decimal
	StablecoinFiatDigits   int32
}

func (a AppConfig) IsLocalEnv() bool {
	return a.EnvName == "local"
}

func SetupAppConfig(app *AppConfig) error {

	viper.AddConfigPath(".")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)

	viper.SetDefault("ENV_NAME", "local")
	viper.SetDefault("PRIME_CALL_TIMEOUT", "10")
	viper.SetDefault("HTTP_CONNECT_TIMEOUT", "5")
	viper.SetDefault("HTTP_CONN_KEEP_ALIVE", "30")
	viper.SetDefault("HTTP_EXPECT_CONTINUE", "1")
	viper.SetDefault("HTTP_IDLE_CONN", "90")
	viper.SetDefault("HTTP_MAX_ALL_IDLE_CONNS", "10")
	viper.SetDefault("HTTP_MAX_HOST_IDLE_CONNS", "5")
	viper.SetDefault("HTTP_RESPONSE_HEADER", "5")
	viper.SetDefault("HTTP_TLS_HANDSHAKE", "5")
	viper.SetDefault("FIAT_CURRENCY_SYMBOL", "USD")
	viper.SetDefault("ORDERS_CACHE_SIZE", "1000")
	viper.SetDefault("CONVERT_SYMBOLS", "usdc")
	viper.SetDefault("TWAP_DURATION", "60")
	viper.SetDefault("TWAP_MIN_NOTIONAL", "100")

	viper.ReadInConfig()

	if err := viper.Unmarshal(&app); err != nil {
		zap.L().Debug("cannot parse env file", zap.Error(err))
	}

	httpClient, err := InitHttpClient(app)
	if err != nil {
		return fmt.Errorf("cannot init the http client %w", err)
	}

	app.HttpClient = httpClient

	app.TwapMaxDiscountPercent = decimal.NewFromFloat32(0.1)
	app.StablecoinFiatDigits = 2

	return nil

}

func (a AppConfig) ConvertSymbols() []string {
	return strings.Split(a.ConvertSymbolsArray, ",")
}

func (a AppConfig) TwapDuration() time.Duration {
	return convertStrIntToDurationOrFatal(a.TwapDurationInMinutes, "TwapDurationInMinutes", time.Minute)
}

func (a AppConfig) HttpConnectTimeout() time.Duration {
	return convertStrIntToDurationOrFatal(a.HttpConnectTimeoutInSeconds, "HttpConnectTimeoutInSeconds", time.Second)
}

func (a AppConfig) PrimeCallTimeout() time.Duration {
	return convertStrIntToDurationOrFatal(a.PrimeCallTimeoutInSeconds, "PrimeCallTimeoutInSeconds", time.Second)
}

func (a AppConfig) HttpConnKeepAlive() time.Duration {
	return convertStrIntToDurationOrFatal(a.HttpConnKeepAliveInSeconds, "HttpConnKeepAliveInSeconds", time.Second)
}

func (a AppConfig) HttpExpectContinue() time.Duration {
	return convertStrIntToDurationOrFatal(a.HttpExpectContinueInSeconds, "HttpExpectContinueInSeconds", time.Second)
}

func (a AppConfig) HttpIdleConn() time.Duration {
	return convertStrIntToDurationOrFatal(a.HttpIdleConnInSeconds, "HttpIdleConnInSeconds", time.Second)
}

func (a AppConfig) HttpResponseHeader() time.Duration {
	return convertStrIntToDurationOrFatal(a.HttpResponseHeaderInSeconds, "HttpResponseHeaderInSeconds", time.Second)
}

func (a AppConfig) OrdersCacheSize() int {
	return convertStrIntOrFatal(a.OrdersCacheSizeInItems, "OrdersCacheSizeInItems")
}

func (a AppConfig) TwapMinNotional() int {
	return convertStrIntOrFatal(a.TwapMinNotionalPerHour, "TwapMinNotionalPerHour")
}

func (a AppConfig) HttpTLSHandshake() time.Duration {
	return convertStrIntToDurationOrFatal(a.HttpTLSHandshakeInSeconds, "HttpTLSHandshakeInSeconds", time.Second)
}

func (a AppConfig) HttpMaxAllIdleConns() int {
	return convertStrIntOrFatal(a.HttpMaxAllIdleConnsCount, "HttpMaxAllIdleConnsCount")
}

func (a AppConfig) HttpMaxHostIdleConns() int {
	return convertStrIntOrFatal(a.HttpMaxHostIdleConnsCount, "HttpMaxHostIdleConnsCount")
}

func convertStrIntToDurationOrFatal(v, n string, dt time.Duration) time.Duration {
	i, err := convertStrIntToDuration(v, dt)
	if err != nil {
		zap.L().Fatal("cannot convert string to int", zap.String("value", v), zap.String("name", n), zap.Error(err))
	}
	return i
}

func convertStrIntOrFatal(v, n string) int {
	i, err := strconv.Atoi(v)
	if err != nil {
		zap.L().Fatal("cannot convert string to int", zap.String("value", v), zap.String("name", n), zap.Error(err))
	}
	return i
}

func convertStrIntToDuration(s string, dt time.Duration) (time.Duration, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return time.Second, err
	}
	return time.Duration(i) * dt, nil
}
