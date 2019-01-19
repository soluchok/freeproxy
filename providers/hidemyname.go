package providers

import (
	"bytes"
	"errors"
	"io"
	"time"

	"github.com/moovweb/gokogiri"
	"github.com/soluchok/go-cloudflare-scraper"
)

type HidemyName struct {
	proxyList  []string
	lastUpdate time.Time
}

func NewHidemyName() *HidemyName {
	return &HidemyName{}
}

func (x *HidemyName) Name() string {
	return "hidemyna.me"
}

func (x *HidemyName) MakeRequest() ([]byte, error) {
	c, err := scraper.NewClient()
	if err != nil {
		return nil, err
	}

	res, err := c.Get("https://hidemyna.me/en/proxy-list")
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	if _, err := io.Copy(&body, res.Body); err != nil {
		return nil, err
	}

	return body.Bytes(), nil
}

func (x *HidemyName) Load(body []byte) ([]string, error) {
	if time.Now().Unix() >= x.lastUpdate.Unix()+(60*15) {
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
