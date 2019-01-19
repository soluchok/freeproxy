package providers

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"
)

type ProxyTech struct {
	proxyList  []string
	lastUpdate time.Time
}

func NewProxyTech() *ProxyTech {
	return &ProxyTech{}
}

func (*ProxyTech) Name() string {
	return "proxy.l337.tech"
}

func (x *ProxyTech) MakeRequest() ([]byte, error) {
	client := &http.Client{Transport: TransportMakeRequest}

	resp, err := client.Get("https://proxy.l337.tech/txt")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body bytes.Buffer
	if _, err := io.Copy(&body, resp.Body); err != nil {
		return nil, err
	}

	return body.Bytes(), nil
}

func (x *ProxyTech) Load() ([]string, error) {
	if time.Now().Unix() >= x.lastUpdate.Unix()+(60*30) {
		x.proxyList = make([]string, 0, 0)
	}

	if len(x.proxyList) != 0 {
		return x.proxyList, nil
	}

	body, err := x.MakeRequest()
	if err != nil {
		return nil, err
	}

	x.proxyList = strings.Split(string(body), "\n")

	x.lastUpdate = time.Now()
	return x.proxyList, nil
}

func (x *ProxyTech) List() ([]string, error) {
	return x.Load()
}
