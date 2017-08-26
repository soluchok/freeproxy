package freeproxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/soluchok/freeproxy/providers"
)

var (
	instance *ProxyGenerator
	once     sync.Once
)

type Provider interface {
	List() ([]string, error)
}

type CheckIP struct {
	IP string
}

type ProxyGenerator struct {
	Timeout   time.Duration
	canLoad   uint32
	providers []Provider
	proxyList chan string
}

func (p *ProxyGenerator) AddProvider(provider Provider) {
	p.providers = append(p.providers, provider)
}

func (p *ProxyGenerator) load() {
	for _, provider := range p.providers {
		ips, err := provider.List()
		if err != nil {
			log.Printf("provider.List() %s", err.Error())
			continue
		}
		for _, proxy := range ips {
			jobs <- proxy
		}
	}
	atomic.StoreUint32(&p.canLoad, 0)
}

func (p *ProxyGenerator) Check(proxy string, transp *http.Transport) bool {
	req, err := http.NewRequest("GET", "http://api.ipify.org/?format=json", nil)
	if err != nil {
		return false
	}
	proxyURL, err := url.Parse(fmt.Sprintf("http://%s", proxy))
	if err != nil {
		return false
	}

	transp.Proxy = http.ProxyURL(proxyURL)
	client := &http.Client{
		//Timeout:   time.Second * p.Timeout,
		Transport: transp,
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
		if atomic.LoadUint32(&p.canLoad) == 0 {
			atomic.StoreUint32(&p.canLoad, 1)
			go p.load()
		}
	}
	return <-p.proxyList
}

func worker(jobs <-chan string, results chan<- string) {
	transp := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: NewProxyGenerator().Timeout * time.Second,
		}).Dial,
		TLSHandshakeTimeout: NewProxyGenerator().Timeout * time.Second,
		DisableKeepAlives:   true,
	}

	for proxy := range jobs {
		if NewProxyGenerator().Check(proxy, transp) {
			results <- proxy
		}
	}
}

var jobs = make(chan string, 500)

func NewProxyGenerator() *ProxyGenerator {
	once.Do(func() {
		instance = &ProxyGenerator{
			Timeout:   5,
			proxyList: make(chan string, 500),
		}
		instance.AddProvider(providers.NewFreeProxyList())
		instance.AddProvider(providers.NewXseoIn())
		for w := 1; w <= 100; w++ {
			go worker(jobs, instance.proxyList)
		}
	})
	return instance
}
