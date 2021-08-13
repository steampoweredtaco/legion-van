package engine

import "net/http"

var (
	httpTransport *http.Transport
	httpClient    *http.Client
)

func init() {
	httpTransport = http.DefaultTransport.(*http.Transport)
	httpClient = http.DefaultClient
}

func SetupHTTP(transport *http.Transport, client *http.Client) {
	httpTransport = transport
	httpClient = client
}
