package freeproxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/soluchok/freeproxy/providers"
)

var (
	instance *ProxyGenerator
	once     sync.Once
	cache    sync.Map
)

type Cache struct {
	time   int64
	result bool
	err    error
}

type Provider interface {
	List() ([]string, error)
}

type CheckIP struct {
	IP string
}

type ProxyGenerator struct {
	canLoad   uint32
	providers []Provider
	proxy     chan string
	job       chan string
}

func (p *ProxyGenerator) isProvider(provider Provider) bool {
	for _, pr := range p.providers {
		if reflect.TypeOf(pr) == reflect.TypeOf(provider) {
			return true
		}
	}
	return false
}

func (p *ProxyGenerator) AddProvider(provider Provider) {
	if !p.isProvider(provider) {
		p.providers = append(p.providers, provider)
	}
}

func (p *ProxyGenerator) load() (errs []error) {
	for _, provider := range p.providers {
		ips, err := provider.List()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, proxy := range ips {
			p.job <- proxy
		}
	}
	atomic.StoreUint32(&p.canLoad, 0)
	return
}

func (p *ProxyGenerator) Check(proxy string) (bool, error) {
	if val, ok := cache.Load(proxy); ok {
		if val.(Cache).time < time.Now().Unix()-60 {
			cache.Delete(proxy)
		} else {
			return val.(Cache).result, val.(Cache).err
		}
	}
	res, err := func() (bool, error) {
		req, err := http.NewRequest("GET", "http://api.ipify.org/?format=json", nil)
		if err != nil {
			return false, err
		}
		proxyURL, err := url.Parse("http://" + proxy)
		if err != nil {
			return false, err
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
			return false, err
		}
		defer resp.Body.Close()

		var body bytes.Buffer
		if _, err := io.Copy(&body, resp.Body); err != nil {
			return false, err
		}

		var checkip CheckIP
		if err := json.Unmarshal(body.Bytes(), &checkip); err != nil {
			return false, err
		}

		return strings.Contains(proxy, checkip.IP), nil
	}()

	cache.Store(proxy, Cache{
		time:   time.Now().Unix(),
		result: res,
		err:    err,
	})
	return res, err
}

func (p *ProxyGenerator) Get() string {
	select {
	case proxy := <-p.proxy:
		return proxy
	case <-time.After(time.Millisecond * 500):
		if atomic.LoadUint32(&p.canLoad) == 0 {
			atomic.StoreUint32(&p.canLoad, 1)
			go p.load()
		}
	}
	return p.Get()
}

func (p *ProxyGenerator) do(proxy string) {
	if ok, _ := p.Check(proxy); ok {
		p.proxy <- proxy
	}
}

func (p *ProxyGenerator) run() {
	for proxy := range p.job {
		go p.do(proxy)
	}
}

func NewProxyGenerator() *ProxyGenerator {
	once.Do(func() {
		instance = &ProxyGenerator{
			proxy: make(chan string),
			job:   make(chan string),
		}

		//add providers to generator
		instance.AddProvider(providers.NewHidemyName())
		instance.AddProvider(providers.NewFreeProxyList())
		instance.AddProvider(providers.NewXseoIn())
		instance.AddProvider(providers.NewFreeProxyListNet())
		instance.AddProvider(providers.NewCoolProxy())

		//run workers
		go instance.run()
	})
	return instance
}
