package freeproxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type checkIP struct {
	IP string
}

func verifyProxy(proxy string) bool {
	req, err := http.NewRequest("GET", "https://api.ipify.org/?format=json", nil)
	if err != nil {
		logrus.Errorf("cannot create new request for verify err:%s", err)
		return false
	}

	proxyURL, err := url.Parse("http://" + proxy)
	if err != nil {
		logrus.Errorf("cannot parse proxy %q err:%s", proxy, err)
		return false
	}

	client := &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          1,
			IdleConnTimeout:       9 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     true,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		logrus.Debugf("cannot verify proxy %q err:%s", proxy, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var body bytes.Buffer
	if _, err := io.Copy(&body, resp.Body); err != nil {
		logrus.Errorf("cannot copy resp.Body err:%s", err)
		return false
	}

	var check checkIP
	if err := json.Unmarshal(body.Bytes(), &check); err != nil {
		logrus.Errorf("%d cannot unmarshal %q to checkIP struct err:%s", resp.StatusCode, body.String(), err)
		return false
	}

	return strings.HasPrefix(proxy, check.IP)
}
