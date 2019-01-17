package main

import (
	"log"

	"github.com/soluchok/freeproxy"
)

func main() {
	gen := freeproxy.New()
	log.Println(gen.Get())
}
