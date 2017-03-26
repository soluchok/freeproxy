package freeproxy

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/soluchok/freeproxy/providers"
)

var instance *ProxyGenerator
var once sync.Once

type Provider interface {
	List() []string
}

type CheckIP struct {
	IP string
}

type ProxyGenerator struct {
	Timeout   time.Duration
	mutex     sync.Mutex
	providers []Provider
	proxyList chan string
}

func (p *ProxyGenerator) AddProvider(provider Provider) {
	p.providers = append(p.providers, provider)
}

func (p *ProxyGenerator) load() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if len(p.proxyList) == 0 {
		for _, provider := range p.providers {
			for _, proxy := range provider.List() {
				if p.Check(proxy) {
					p.proxyList <- proxy
				}
			}
		}
	}
}

func (p *ProxyGenerator) Check(proxy string) bool {
	req, err := http.NewRequest("GET", "http://api.ipify.org/?format=json", nil)
	if err != nil {
		return false
	}
	proxyURL, err := url.Parse(fmt.Sprintf("http://%s", proxy))
	if err != nil {
		return false
	}
	client := &http.Client{
		Timeout: time.Duration(p.Timeout * time.Second),
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy: http.ProxyURL(proxyURL)},
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	var checkip CheckIP
	err = json.Unmarshal(body, &checkip)
	if err != nil {
		return false
	}
	return strings.Contains(proxy, checkip.IP)
}

func (p *ProxyGenerator) Get() string {
	select {
	case proxy := <-p.proxyList:
		return proxy
	default:
		go p.load()
	}
	return <-p.proxyList
}

func NewProxyGenerator() *ProxyGenerator {
	once.Do(func() {
		instance = &ProxyGenerator{
			Timeout:   5,
			proxyList: make(chan string, 10000),
		}
		instance.AddProvider(providers.NewFreeProxyListNet())
		instance.AddProvider(providers.NewXseoIn())
	})
	return instance
}
