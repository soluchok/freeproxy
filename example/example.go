package main

import (
	"fmt"

	"github.com/soluchok/freeproxy"
)

func main() {
	gen := freeproxy.New()
	fmt.Println(gen.Get())
}
