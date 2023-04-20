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
		&Call{
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

func makeCall(ctx context.Context, call *Call) *Response {

	response := &Response{
		Call: call,
	}

	if strings.ToLower(call.HttpMethod) != "get" && strings.ToLower(call.HttpMethod) != "post" {
		response.Error = fmt.Errorf("prime.MakeCall HttpMethod must get GET or POST - received: %s", call.HttpMethod)
		return response
	}

	method := "POST"
	if strings.ToLower(call.HttpMethod) == "get" {
		method = "GET"
	}

	parsedUrl, err := url.Parse(call.Url)
	if err != nil {
		response.Error = fmt.Errorf("Unable to parse Call URL: %s - msg: %v", call.Url, err)
		return response
	}

	log.WithFields(log.Fields{
		"url":         call.Url,
		"method":      call.HttpMethod,
		"requestBody": string(call.Body),
		"state":       "beforePrimeCall",
	}).Debug("prime.MakeCall")

	t := time.Now().Unix()

	req, err := http.NewRequestWithContext(ctx, method, call.Url, bytes.NewReader(call.Body))
	if err != nil {
		response.Error = fmt.Errorf("unable to to create HTTP request: %s - msg: %v", call.Url, err)
		return response
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-CB-ACCESS-KEY", call.Credentials.AccessKey)
	req.Header.Add("X-CB-ACCESS-PASSPHRASE", call.Credentials.Passphrase)
	req.Header.Add("X-CB-ACCESS-SIGNATURE", sign(parsedUrl.Path, string(call.Body), method, call.Credentials.SigningKey, t))
	req.Header.Add("X-CB-ACCESS-TIMESTAMP", fmt.Sprintf("%d", t))

	client := http.Client{Transport: GetHttpTransport()}

	res, err := client.Do(req)
	if err != nil {
		response.Error = fmt.Errorf("unable call to URL: %s - msg: %v", call.Url, err)
		return response
	}

	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		response.Error = fmt.Errorf("unable to read response body: %s - err: %w", call.Url, err)
		return response
	}

	log.WithFields(log.Fields{
		"url":           call.Url,
		"method":        call.HttpMethod,
		"httpStatus":    res.StatusCode,
		"httpStatusMsg": res.Status,
		"requestBody":   string(call.Body),
		"responseBody":  string(resBody),
		"state":         "afterPrimeCall",
	}).Debugf("prime.MakeCall")

	response.Body = resBody
	response.HttpStatusCode = res.StatusCode
	response.HttpStatusMsg = res.Status

	if call.ExpectedHttpStatusCode > 0 && res.StatusCode != call.ExpectedHttpStatusCode {

		var errMsg ErrorMessage
		if strings.Contains(string(response.Body), "message") {
			_ = json.Unmarshal(response.Body, &errMsg)
		}

		response.Error = fmt.Errorf(
			"Expected status code: %d - received: %d - status msg: %s - url %s - response: %s - repsonse msg: %s",
			call.ExpectedHttpStatusCode,
			res.StatusCode,
			res.Status,
			call.Url,
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
