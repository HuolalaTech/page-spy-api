package util

import "net"

// GetLocalIP 获取本机ip 获取失败返回 ""
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return ""
	}

	for _, address := range addrs {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}

	return ""
}

func GetLocalIPList() []string {
	addrs, err := net.InterfaceAddrs()

	arr := []string{}
	if err != nil {
		return arr
	}

	for _, address := range addrs {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				arr = append(arr, ipNet.IP.String())
			}
		}
	}

	return arr
}
