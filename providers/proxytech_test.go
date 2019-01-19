package providers

import (
	"testing"
)

func TestProxyTech_List(t *testing.T) {
	ips, err := (&ProxyTech{}).List()
	if err != nil {
		t.Error(err)
	}

	if len(ips) < 50 {
		t.Error(err)
	}
}
