## Go (golang) library for free proxy.
## Installation

To install freeproxy, simply run:
```
$ go get github.com/soluchok/freeproxy
```
## Example
```go
package main

import (
    "log"
    "github.com/soluchok/freeproxy"
)

func main() {
    gen := freeproxy.NewProxyGenerator()
    log.Println(gen.Get())
}

//190.248.128.122:3128
```
