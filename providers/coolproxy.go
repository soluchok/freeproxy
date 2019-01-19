package providers

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/moovweb/gokogiri"
)

const coolProxyURL = `https://www.cool-proxy.net/proxies/http_proxy_list/sort:score/direction:desc`

type CoolProxy struct {
	proxyList  []string
	lastUpdate time.Time
}

func NewCoolProxy() *CoolProxy {
	return new(CoolProxy)
}

// TODO: need implementation
func (*CoolProxy) SetProxy(_ string) {}

func (*CoolProxy) Name() string {
	return "www.cool-proxy.net"
}

func (c *CoolProxy) Load(body []byte) ([]string, error) {
	if time.Now().Unix() >= c.lastUpdate.Unix()+(60*10) {
		c.proxyList = make([]string, 0, 0)
	}

	if len(c.proxyList) != 0 {
		return c.proxyList, nil
	}

	if body == nil {
		var err error
		if body, err = c.MakeRequest(); err != nil {
			return nil, err
		}
	}

	doc, err := gokogiri.ParseHtml(body)
	if err != nil {
		return nil, err
	}

	defer doc.Free()

	ips, err := doc.Search(`//*[@id="main"]/table/tr/td[1][not(@colspan)]/script`)
	if err != nil {
		return nil, err
	}

	ports, err := doc.Search(`//*[@id="main"]/table/tr/td[2]`)
	if err != nil {
		return nil, err
	}

	if len(ips) == 0 {
		return nil, errors.New("ip not found")
	}

	if len(ips) != len(ports) {
		return nil, errors.New("len port not equal ip")
	}

	r := regexp.MustCompile(`"(.*?[^\\])"`)

	for i, ip := range ips {
		raw := r.FindStringSubmatch(ip.Content())
		if len(raw) != 2 {
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(string(bytes.Map(rot13, []byte(raw[1]))))
		if err != nil {
			continue
		}

		c.proxyList = append(c.proxyList, fmt.Sprintf("%s:%s", decoded, ports[i].Content()))
	}
	c.lastUpdate = time.Now()
	return c.proxyList, nil
}

func (c *CoolProxy) MakeRequest() ([]byte, error) {
	var (
		body   bytes.Buffer
		client = &http.Client{Timeout: time.Second * 10, Transport: &http.Transport{
			DisableKeepAlives: true,
		}}
	)

	req, err := http.NewRequest(http.MethodGet, coolProxyURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	return body.Bytes(), err
}

func (c *CoolProxy) List() ([]string, error) {
	return c.Load(nil)
}

func rot13(x rune) rune {
	capital := x >= 'A' && x <= 'Z'
	if !capital && (x < 'a' || x > 'z') {
		return x
	}
	x += 13
	if capital && x > 'Z' || !capital && x > 'z' {
		x -= 26
	}
	return x
}
