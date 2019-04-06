package freeproxy

import (
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"github.com/soluchok/freeproxy/providers"
)

var (
	instance  *ProxyGenerator
	usedProxy sync.Map
	once      sync.Once
)

type Verify func(proxy string) bool

type ProxyGenerator struct {
	lastValidProxy string
	cache          *cache.Cache
	VerifyFn       Verify
	providers      []Provider
	proxy          chan string
	job            chan string
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

func shuffle(vals []string) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(vals) > 0 {
		n := len(vals)
		randIndex := r.Intn(n)
		vals[n-1], vals[randIndex] = vals[randIndex], vals[n-1]
		vals = vals[:n-1]
	}
}

func (p *ProxyGenerator) load() {
	for {
		for _, provider := range p.providers {
			usedProxy.Store(p.lastValidProxy, time.Now().Hour())
			provider.SetProxy(p.lastValidProxy)

			ips, err := provider.List()
			if err != nil {
				p.lastValidProxy = ""
				logrus.Errorf("cannot load list of proxy %s err:%s", provider.Name(), err)
				continue
			}

			logrus.Println(provider.Name(), len(ips))

			usedProxy.Range(func(key, value interface{}) bool {
				if value.(int) != time.Now().Hour() {
					usedProxy.Delete(key)
				}
				return true
			})

			logrus.Debugf("provider %s found ips %d", provider.Name(), len(ips))
			shuffle(ips)
			for _, proxy := range ips {
				p.job <- proxy
			}
		}
	}
}

func (p *ProxyGenerator) Get() string {
	proxy := <-p.proxy
	_, ok := usedProxy.Load(proxy)
	if !ok {
		p.lastValidProxy = proxy
	}
	return proxy
}

func (p *ProxyGenerator) verifyWithCache(proxy string) bool {
	val, found := p.cache.Get(proxy)
	if found {
		return val.(bool)
	}
	res := p.VerifyFn(proxy)
	p.cache.Set(proxy, res, cache.DefaultExpiration)
	return res
}

func (p *ProxyGenerator) do(proxy string) {
	if p.verifyWithCache(proxy) {
		p.proxy <- proxy
	}
}

func (p *ProxyGenerator) worker() {
	for proxy := range p.job {
		p.do(proxy)
	}
}

func (p *ProxyGenerator) run() {
	go p.load()

	for w := 1; w <= 40; w++ {
		go p.worker()
	}
}

func New() *ProxyGenerator {
	once.Do(func() {
		instance = &ProxyGenerator{
			cache:    cache.New(20*time.Minute, 5*time.Minute),
			VerifyFn: verifyProxy,
			proxy:    make(chan string),
			job:      make(chan string, 100),
		}

		//add providers to generator
		instance.AddProvider(providers.NewFreeProxyList())
		instance.AddProvider(providers.NewHidemyName())
		instance.AddProvider(providers.NewXseoIn())
		instance.AddProvider(providers.NewFreeProxyListNet())
		instance.AddProvider(providers.NewCoolProxy())
		//instance.AddProvider(providers.NewProxyTech())
		instance.AddProvider(providers.NewPubProxy())
		instance.AddProvider(providers.NewProxyList())
		//run workers
		go instance.run()
	})
	return instance
}
