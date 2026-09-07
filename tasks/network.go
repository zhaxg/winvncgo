package tasks

import (
	"net"
	"strings"
)

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	var fallback string

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		ip := ipNet.IP.String()

		// 跳过链路本地地址 (169.254.x.x)
		if strings.HasPrefix(ip, "169.254.") {
			continue
		}

		// 优先返回局域网地址 (192.168.x.x / 10.x.x.x / 172.16-31.x.x)
		if strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") {
			return ip
		}

		// 记住第一个非链路本地地址作为备选
		if fallback == "" {
			fallback = ip
		}
	}

	if fallback != "" {
		return fallback
	}
	return "127.0.0.1"
}

// getAllIPs 返回所有有效 IPv4 地址
func getAllIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		ip := ipNet.IP.String()
		if strings.HasPrefix(ip, "169.254.") {
			continue
		}
		ips = append(ips, ip)
	}
	return ips
}
