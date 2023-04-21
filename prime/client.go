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
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type apiRequest struct {
	Url                    string
	HttpMethod             string
	Body                   []byte
	ExpectedHttpStatusCode int
	Credentials            *Credentials
}

type apiResponse struct {
	Request        *apiRequest
	Body           []byte
	HttpStatusCode int
	HttpStatusMsg  string
	Error          error
}

func (r apiResponse) IsHttpOk() bool {
	return r.HttpStatusCode == 200
}

func (r apiResponse) Unmarshal(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

func PrimePost(
	ctx context.Context,
	url string,
	request,
	response interface{},
) error {
	return primeCall(ctx, url, http.MethodPost, http.StatusOK, request, response)
}

func PrimeGet(
	ctx context.Context,
	url string,
	request,
	response interface{},
) error {
	return primeCall(ctx, url, http.MethodGet, http.StatusOK, request, response)
}

func primeCall(
	ctx context.Context,
	url,
	httpMethod string,
	expectedHttpStatusCode int,
	request,
	response interface{},
) error {

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("unable to marshal prime request: %w", err)
	}

	resp := makeCall(
		ctx,
		&apiRequest{
			Url:                    url,
			HttpMethod:             httpMethod,
			Body:                   body,
			ExpectedHttpStatusCode: expectedHttpStatusCode,
			Credentials:            GetCredentials(),
		},
	)

	if resp.Error != nil {
		return fmt.Errorf("prime error: %w", err)
	}

	if err := resp.Unmarshal(response); err != nil {
		return fmt.Errorf("unable to marshall prime response: %w", err)
	}

	return nil
}

func makeCall(ctx context.Context, request *apiRequest) *apiResponse {

	response := &apiResponse{
		Request: request,
	}

	if strings.ToLower(request.HttpMethod) != "get" && strings.ToLower(request.HttpMethod) != "post" {
		response.Error = fmt.Errorf("prime.MakeCall HttpMethod must get GET or POST - received: %s", request.HttpMethod)
		return response
	}

	method := http.MethodPost
	if strings.ToLower(request.HttpMethod) == "get" {
		method = http.MethodGet
	}

	parsedUrl, err := url.Parse(request.Url)
	if err != nil {
		response.Error = fmt.Errorf("cannot parse URL: %s - msg: %v", request.Url, err)
		return response
	}

	log.WithFields(log.Fields{
		"url":         request.Url,
		"method":      request.HttpMethod,
		"requestBody": string(request.Body),
		"state":       "beforePrimeCall",
	}).Debug("prime.MakeCall")

	t := time.Now().Unix()

	req, err := http.NewRequestWithContext(ctx, method, request.Url, bytes.NewReader(request.Body))
	if err != nil {
		response.Error = fmt.Errorf("cannot create HTTP request: %s - msg: %v", request.Url, err)
		return response
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-CB-ACCESS-KEY", request.Credentials.AccessKey)
	req.Header.Add("X-CB-ACCESS-PASSPHRASE", request.Credentials.Passphrase)
	req.Header.Add("X-CB-ACCESS-SIGNATURE", sign(parsedUrl.Path, string(request.Body), method, request.Credentials.SigningKey, t))
	req.Header.Add("X-CB-ACCESS-TIMESTAMP", fmt.Sprintf("%d", t))

	client := http.Client{Transport: GetHttpTransport()}

	res, err := client.Do(req)
	if err != nil {
		response.Error = fmt.Errorf("cannot call URL: %s - msg: %v", request.Url, err)
		return response
	}

	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		response.Error = fmt.Errorf("cannot read response body: %s - err: %w", request.Url, err)
		return response
	}

	log.WithFields(log.Fields{
		"url":           request.Url,
		"method":        request.HttpMethod,
		"httpStatus":    res.StatusCode,
		"httpStatusMsg": res.Status,
		"requestBody":   string(request.Body),
		"responseBody":  string(resBody),
		"state":         "afterPrimeCall",
	}).Debugf("prime.MakeCall")

	response.Body = resBody
	response.HttpStatusCode = res.StatusCode
	response.HttpStatusMsg = res.Status

	if request.ExpectedHttpStatusCode > 0 && res.StatusCode != request.ExpectedHttpStatusCode {

		var errMsg ErrorMessage
		if strings.Contains(string(response.Body), "message") {
			_ = json.Unmarshal(response.Body, &errMsg)
		}

		response.Error = fmt.Errorf(
			"expected status code: %d - received: %d - status msg: %s - url %s - response: %s - repsonse msg: %s",
			request.ExpectedHttpStatusCode,
			res.StatusCode,
			res.Status,
			request.Url,
			string(resBody),
			errMsg.Message,
		)
	}

	return response
}

func sign(path, body, method, signingKey string, t int64) string {
	h := hmac.New(sha256.New, []byte(signingKey))
	h.Write([]byte(fmt.Sprintf("%d%s%s%s", t, method, path, body)))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
