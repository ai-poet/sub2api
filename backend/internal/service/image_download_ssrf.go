package service

import (
	"context"
	"net"
	"strings"
)

// Image downloads must never reach loopback, private networks or cloud metadata.
// Keep these helpers independent from the optional channel-monitor feature.
var imageDownloadBlockedHostnames = map[string]struct{}{
	"localhost":                  {},
	"localhost.localdomain":      {},
	"metadata":                   {},
	"metadata.google.internal":   {},
	"metadata.goog":              {},
	"instance-data":              {},
	"instance-data.ec2.internal": {},
}

var imageDownloadBlockedCIDRs = mustParseImageDownloadCIDRs([]string{
	"127.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"100.64.0.0/10",
	"0.0.0.0/8",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
	"::/128",
})

func mustParseImageDownloadCIDRs(cidrs []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic("image download SSRF: invalid CIDR " + cidr + ": " + err.Error())
		}
		out = append(out, network)
	}
	return out
}

func isBlockedHostname(hostname string) bool {
	if hostname == "" {
		return true
	}
	_, blocked := imageDownloadBlockedHostnames[strings.ToLower(hostname)]
	return blocked
}

func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsUnspecified() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
		return true
	}
	for _, network := range imageDownloadBlockedCIDRs {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func isPrivateOrLoopbackHost(ctx context.Context, hostname string) (bool, error) {
	if isBlockedHostname(hostname) {
		return true, nil
	}
	if ip := net.ParseIP(hostname); ip != nil {
		return isPrivateIP(ip), nil
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return false, err
	}
	if len(addresses) == 0 {
		return true, nil
	}
	for _, address := range addresses {
		if isPrivateIP(address.IP) {
			return true, nil
		}
	}
	return false, nil
}
