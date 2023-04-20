package prime

import (
	"net/http"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

var httpTransport *http.Transport

func SetHttpTransport(tr *http.Transport) *http.Transport {
	httpTransport = tr
	return httpTransport
}

func GetHttpTransport() *http.Transport {
	return httpTransport
}

func InitHttpTransPort() *http.Transport {
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
