package proxy

import (
	"fmt"
	"net/http/httputil"
	"net/url"

	"github.com/HuolalaTech/page-spy-api/rpc"
	"github.com/labstack/echo/v4"
)

type ProxyManager struct {
	proxies        map[string]*httputil.ReverseProxy
	addressManager *rpc.AddressManager
}

func (pm *ProxyManager) Proxy(machineId string, c echo.Context) error {
	p, ok := pm.proxies[machineId]
	if !ok {
		return fmt.Errorf("get proxy by machineId %s not found", machineId)
	}

	c.Request().Host = pm.getProxyHost(machineId)
	p.ServeHTTP(c.Response(), c.Request())
	return nil
}

func (pm *ProxyManager) getProxyHost(findId string) string {
	for machineId, address := range pm.addressManager.GetMachineIpInfo() {
		if machineId == findId {
			return fmt.Sprintf("http://%s:%s", address.Ip, address.Port)
		}
	}
	return ""
}

func NewProxy(addressManager *rpc.AddressManager) (*ProxyManager, error) {
	proxies := make(map[string]*httputil.ReverseProxy)

	for machineId, address := range addressManager.GetMachineIpInfo() {
		u := fmt.Sprintf("http://%s:%s", address.Ip, address.Port)
		proxyURL, err := url.Parse(u)
		if err != nil {
			return nil, fmt.Errorf("parse url %s error", u)
		}

		reverseProxy := httputil.NewSingleHostReverseProxy(proxyURL)
		proxies[machineId] = reverseProxy
	}

	return &ProxyManager{
		proxies:        proxies,
		addressManager: addressManager,
	}, nil
}
