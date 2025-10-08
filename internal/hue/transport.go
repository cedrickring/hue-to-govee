package hue

import (
	"crypto/tls"
	"net/http"
)

// hueTransport is a http.RoundTripper that adds the Hue application key to the request headers.
type hueTransport struct {
	hueUsername string

	T http.RoundTripper
}

// newHueTransport creates a new hueTransport with the given hueUsername.
func newHueTransport(hueUsername string) *hueTransport {
	return &hueTransport{
		hueUsername: hueUsername,
		T: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // Skip TLS verification for local bridge
			},
		},
	}
}

func (t *hueTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("hue-application-key", t.hueUsername)
	return t.T.RoundTrip(req)
}
