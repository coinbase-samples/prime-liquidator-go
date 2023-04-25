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
	"encoding/json"
	"fmt"
	"os"
)

var credentials *Credentials

func GetCredentials() *Credentials {
	return credentials
}

func SetCredentials(c *Credentials) {
	credentials = c
}

func InitCredentials() (*Credentials, error) {

	if credentials != nil {
		return credentials, nil
	}

	credentials = &Credentials{}
	if err := json.Unmarshal([]byte(os.Getenv("PRIME_CREDENTIALS")), credentials); err != nil {
		return nil, fmt.Errorf("Failed to deserialize prime credentials JSON: %w", err)
	}

	return credentials, nil
}
