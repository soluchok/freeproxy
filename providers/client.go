package providers

import (
	"net"
	"net/http"
	"time"
)

func NewTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   9 * time.Second,
			KeepAlive: 9 * time.Second,
			DualStack: true,
		}).DialContext,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 3 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     true,
	}
}

func NewClient() *http.Client {
	return &http.Client{
		Timeout:   time.Second * 20,
		Transport: NewTransport(),
	}
}
