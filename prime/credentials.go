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
