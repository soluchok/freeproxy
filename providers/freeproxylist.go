package providers

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/moovweb/gokogiri"
)

type FreeProxyList struct{}

func NewFreeProxyList() *FreeProxyList {
	return &FreeProxyList{}
}

func (x *FreeProxyList) MakeRequest() ([]byte, error) {
	req, err := http.NewRequest("GET", "https://free-proxy-list.net/", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept-Language", "en-US,en;q=0.8,uk;q=0.6,ru;q=0.4")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Authority", "free-proxy-list.net")
	req.Header.Set("Referer", "https://free-proxy-list.net/web-proxy.html")

	resp, err := http.DefaultClient.Do(req)
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

func (x *FreeProxyList) Load(body []byte) ([]string, error) {
	if body == nil {
		var err error
		if body, err = x.MakeRequest(); err != nil {
			return nil, err
		}
	}

	doc, err := gokogiri.ParseHtml(body)
	if err != nil {
		return nil, err
	}
	defer doc.Free()

	ips, err := doc.Search(`//*[@id="proxylisttable"]/tbody/tr/td[1]`)
	if err != nil {
		return nil, err
	}
	ports, err := doc.Search(`//*[@id="proxylisttable"]/tbody/tr/td[2]`)
	if err != nil {
		return nil, err
	}

	if len(ips) == 0 {
		return nil, errors.New("ip not found")
	}

	if len(ips) != len(ports) {
		return nil, errors.New("len port not equal ip")
	}

	proxyList := make([]string, 0, len(ips))

	for i, ip := range ips {
		proxyList = append(proxyList, ip.Content()+":"+ports[i].Content())
	}

	return proxyList, nil
}

func (x *FreeProxyList) List() ([]string, error) {
	return x.Load(nil)
}
