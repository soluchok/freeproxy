package providers

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/moovweb/gokogiri"
)

var TransportMakeRequest = &http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}).DialContext,
	MaxIdleConns:          10,
	IdleConnTimeout:       9 * time.Second,
	TLSHandshakeTimeout:   5 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	DisableKeepAlives:     true,
}

type FreeProxyList struct {
	proxyList  []string
	lastUpdate time.Time
}

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

	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: TransportMakeRequest,
	}

	resp, err := client.Do(req)
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
	if time.Now().Unix() >= x.lastUpdate.Unix()+(60*10) {
		x.proxyList = make([]string, 0, 0)
	}

	if len(x.proxyList) != 0 {
		return x.proxyList, nil
	}

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

	x.proxyList = make([]string, 0, len(ips))

	for i, ip := range ips {
		x.proxyList = append(x.proxyList, ip.Content()+":"+ports[i].Content())
	}

	x.lastUpdate = time.Now()
	return x.proxyList, nil
}

func (x *FreeProxyList) List() ([]string, error) {
	defer runtime.GC()
	return x.Load(nil)
}
