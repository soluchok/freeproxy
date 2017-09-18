package freeproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
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
	canLoad     uint32
	providers   []Provider
	proxyList   chan string
	workerCount int
	jobs        chan string
}

func (p *ProxyGenerator) AddProvider(provider Provider) {
	p.providers = append(p.providers, provider)
}

func (p *ProxyGenerator) load() {
	for _, provider := range p.providers {
		ips, err := provider.List()
		if err != nil {
			logrus.Error(err)
			continue
		}
		for _, proxy := range ips {
			p.jobs <- proxy
		}
	}
	atomic.StoreUint32(&p.canLoad, 0)
}

func (p *ProxyGenerator) Check(proxy string) bool {
	req, err := http.NewRequest("GET", "http://api.ipify.org/?format=json", nil)
	if err != nil {
		logrus.Error(err)
		return false
	}
	proxyURL, err := url.Parse(fmt.Sprintf("http://%s", proxy))
	if err != nil {
		logrus.Error(err)
		return false
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
		logrus.Error(err)
		return false
	}
	defer resp.Body.Close()

	var body bytes.Buffer
	if _, err := io.Copy(&body, resp.Body); err != nil {
		logrus.Error(err)
		return false
	}

	var checkip CheckIP
	if err := json.Unmarshal(body.Bytes(), &checkip); err != nil {
		logrus.Error(err)
		return false
	}

	return strings.Contains(proxy, checkip.IP)
}

func (p *ProxyGenerator) Get() string {
	select {
	case proxy := <-p.proxyList:
		return proxy
	case <-time.After(time.Second * 1):
		if atomic.LoadUint32(&p.canLoad) == 0 {
			atomic.StoreUint32(&p.canLoad, 1)
			go p.load()
		}
	}
	return p.Get()
}

func (p *ProxyGenerator) NumWorker() int {
	if p.workerCount <= 0 {
		return runtime.NumCPU() * 2
	}
	return p.workerCount
}

func worker(jobs <-chan string, results chan<- string) {
	for proxy := range jobs {
		if NewProxyGenerator().Check(proxy) {
			results <- proxy
		}
	}
}

func NewProxyGenerator() *ProxyGenerator {
	once.Do(func() {
		instance = &ProxyGenerator{
			proxyList: make(chan string, 500),
			jobs:      make(chan string, 500),
		}

		//add providers to generator
		instance.AddProvider(providers.NewFreeProxyList())
		instance.AddProvider(providers.NewXseoIn())
		instance.AddProvider(providers.NewFreeProxyListNet())

		logrus.Infof("Start %d workers ...", instance.NumWorker())

		//run workers
		for w := 1; w <= instance.NumWorker(); w++ {
			go worker(instance.jobs, instance.proxyList)
		}
	})
	return instance
}
