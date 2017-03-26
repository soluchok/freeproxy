package providers

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	xmlpath "gopkg.in/xmlpath.v2"
)

type FreeProxyListNet struct {
	jsFileName string
	mutex      sync.Mutex
	wasUpdate  uint32
	proxyList  []string
}

var instanceFreeProxyListNet *FreeProxyListNet
var instanceFreeProxyListNetOnce sync.Once

func NewFreeProxyListNet() *FreeProxyListNet {
	instanceFreeProxyListNetOnce.Do(func() {
		instanceFreeProxyListNet = (&FreeProxyListNet{jsFileName: "FreeProxyListNet.js"}).init()
	})
	return instanceFreeProxyListNet
}

func (x *FreeProxyListNet) init() *FreeProxyListNet {
	ticker := time.NewTicker(time.Minute * 15)
	go func() {
		for _ = range ticker.C {
			x.mutex.Lock()
			atomic.StoreUint32(&x.wasUpdate, 0)
			x.mutex.Unlock()
		}
	}()
	return x
}

func (x *FreeProxyListNet) jsFileContent() string {
	return `
	var webPage = require('webpage');
	var page = webPage.create();

	page.open('https://free-proxy-list.net/', function() {
		page.evaluate(function() {    
			$('#proxylisttable_length select').val('80').change();
		})
		setTimeout(function() {
	        console.log(page.content);
			phantom.exit();
		}, 2000)
	});`
}

func (x *FreeProxyListNet) load() []string {
	var proxyList []string

	file, err := os.Create(x.jsFileName)
	if err != nil {
		log.Println(err)
		return proxyList
	}
	file.WriteString(x.jsFileContent())

	out, err := exec.Command("phantomjs", x.jsFileName).Output()
	if err != nil {
		log.Println(err)
		return proxyList
	}
	reader := strings.NewReader(string(out))
	xmlroot, err := xmlpath.ParseHTML(reader)
	if err != nil {
		log.Println(err)
		return proxyList
	}
	obj := xmlpath.MustCompile(`//td[1]`)
	objPorts := xmlpath.MustCompile(`//td[2]`)
	itPorts := objPorts.Iter(xmlroot)
	it := obj.Iter(xmlroot)

	for it.Next() && itPorts.Next() {
		ip := it.Node().String()
		port := itPorts.Node().String()
		isIP, _ := regexp.MatchString(`(\d{1,3}\.){3}\d{1,3}`, ip)
		if isIP {
			proxyList = append(proxyList, fmt.Sprintf("%s:%s", ip, port))
		}
	}
	return proxyList
}

func (x *FreeProxyListNet) List() []string {
	if atomic.LoadUint32(&x.wasUpdate) == 1 {
		return x.proxyList
	}
	x.mutex.Lock()
	defer x.mutex.Unlock()

	if x.wasUpdate == 0 {
		x.proxyList = x.load()
		atomic.StoreUint32(&x.wasUpdate, 1)
	}
	return x.proxyList
}
