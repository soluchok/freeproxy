package freeproxy

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/soluchok/freeproxy/providers"
)

var (
	instance *ProxyGenerator
	once     sync.Once
	cache    sync.Map
)

type Verify func(proxy string) bool

type Cache struct {
	time   int64
	result bool
	err    error
}

type ProxyGenerator struct {
	VerifyFn  Verify
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

func (p *ProxyGenerator) load() {
	for _, provider := range p.providers {
		ips, err := provider.List()
		if err != nil {
			logrus.Errorf("cannot load list of proxy %s err:%s", provider.Name(), err)
			continue
		}
		logrus.Infof("provider %s found ips %d", provider.Name(), len(ips))
		for _, proxy := range ips {
			p.job <- proxy
		}
	}
	atomic.StoreUint32(&p.canLoad, 0)
	return
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
	if p.VerifyFn(proxy) {
		p.proxy <- proxy
	}
}

func (p *ProxyGenerator) worker() {
	for proxy := range p.job {
		p.do(proxy)
	}
}

func (p *ProxyGenerator) run() {
	for w := 1; w <= 30; w++ {
		go p.worker()
	}
}

func New() *ProxyGenerator {
	once.Do(func() {
		instance = &ProxyGenerator{
			VerifyFn: verifyProxy,
			proxy:    make(chan string),
			job:      make(chan string),
		}

		//add providers to generator
		instance.AddProvider(providers.NewHidemyName())
		instance.AddProvider(providers.NewFreeProxyList())
		instance.AddProvider(providers.NewXseoIn())
		instance.AddProvider(providers.NewFreeProxyListNet())
		instance.AddProvider(providers.NewCoolProxy())
		instance.AddProvider(providers.NewProxyTech())
		//run workers
		go instance.run()
	})
	return instance
}
