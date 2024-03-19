package proxy

import (
	"fmt"
	"net/http/httputil"
	"net/url"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/rpc"
	"github.com/labstack/echo/v4"
)

type proxyInfo struct {
	host  string
	proxy *httputil.ReverseProxy
}

type ProxyManager struct {
	info           map[string]*proxyInfo
	addressManager *rpc.AddressManager
}

func (pm *ProxyManager) Proxy(machineId string, c echo.Context) error {
	info, ok := pm.info[machineId]
	if !ok {
		return fmt.Errorf("get proxy by machineId %s not found", machineId)
	}

	c.Request().Host = info.host
	info.proxy.ServeHTTP(c.Response(), c.Request())
	return nil
}

func NewProxy(config *config.Config, addressManager *rpc.AddressManager) (*ProxyManager, error) {
	proxies := make(map[string]*proxyInfo)

	for machineId, address := range addressManager.GetMachineIpInfo() {
		host := fmt.Sprintf("%s:%s", address.Ip, config.Port)
		u := fmt.Sprintf("http://%s", host)
		proxyURL, err := url.Parse(u)
		if err != nil {
			return nil, fmt.Errorf("parse url %s error", u)
		}

		reverseProxy := httputil.NewSingleHostReverseProxy(proxyURL)
		proxies[machineId] = &proxyInfo{
			host:  host,
			proxy: reverseProxy,
		}
	}

	return &ProxyManager{
		info:           proxies,
		addressManager: addressManager,
	}, nil
}
