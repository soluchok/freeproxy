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

type XseoIn struct {
	jsFileName string
	mutex      sync.Mutex
	wasUpdate  uint32
	proxyList  []string
}

var instanceXseoIn *XseoIn
var instanceXseoInOnce sync.Once

func NewXseoIn() *XseoIn {
	instanceXseoInOnce.Do(func() {
		instanceXseoIn = (&XseoIn{jsFileName: "XseoIn.js"}).init()
	})
	return instanceXseoIn
}

func (x *XseoIn) init() *XseoIn {
	ticker := time.NewTicker(time.Minute * 30)
	go func() {
		for _ = range ticker.C {
			x.mutex.Lock()
			atomic.StoreUint32(&x.wasUpdate, 0)
			x.mutex.Unlock()
		}
	}()
	return x
}

func (x *XseoIn) jsFileContent() string {
	return `
	var webPage = require('webpage');
	var page = webPage.create();
	var postBody = 'submit=Показать по 150 прокси на странице';

	page.open('http://xseo.in/proxylist','POST',postBody, function() {
		page.evaluate(function() {    
			$('.submit').click()
		})
		setTimeout(function() {
	        console.log(page.content);
			phantom.exit();
		}, 5000)
	});`
}

func (x *XseoIn) load() []string {
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
	obj := xmlpath.MustCompile("/html/body/table[1]/tbody/tr/td[1]/table[2]/tbody/tr/td/table[1]/tbody/tr/td[1]/font/text()")
	it := obj.Iter(xmlroot)
	for it.Next() {
		ip := it.Node().String()
		isIP, _ := regexp.MatchString(`(\d{1,3}\.){3}\d{1,3}`, ip)
		if isIP {
			it.Next()
			proxyList = append(proxyList, fmt.Sprintf("%s:%s", ip, it.Node().String()))
		}
	}
	return proxyList
}

func (x *XseoIn) List() []string {
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
