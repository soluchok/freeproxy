package providers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/moovweb/gokogiri"
)

type HidemyName struct {
	proxyList  []string
	lastUpdate time.Time
}

func NewHidemyName() *HidemyName {
	return &HidemyName{}
}

func (x *HidemyName) MakeRequest() ([]byte, error) {
	req, err := http.NewRequest("GET", "https://hidemy.name/ua/proxy-list/", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept-Language", "en-US,en;q=0.8,uk;q=0.6,ru;q=0.4")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Referer", "https://hidemy.name/ua/proxy-list/?start=128")
	req.Header.Set("Authority", "hidemy.name")

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

func (x *HidemyName) Load(body []byte) ([]string, error) {
	if time.Now().Unix() >= x.lastUpdate.Unix()+(60*5) {
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

	ips, err := doc.Search(`//td[contains(@class, 'tdl')]`)
	if err != nil {
		return nil, err
	}

	if len(ips) == 0 {
		return nil, errors.New("ip not found")
	}

	x.proxyList = make([]string, 0, len(ips))

	for _, ip := range ips {
		port := ip.NextSibling()
		if ipRegexp.MatchString(ip.Content()) {
			x.proxyList = append(x.proxyList, ip.Content()+":"+port.Content())
		}

	}
	x.lastUpdate = time.Now()
	return x.proxyList, nil
}

func (x *HidemyName) List() ([]string, error) {
	return x.Load(nil)
}
