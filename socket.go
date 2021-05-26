package dproxy

import (
	"golang.org/x/net/context"
	"net"
)

func (self *Server) ServerSocket(ctx context.Context, network, addr string) (net.Conn, error) {

	host, _, _ := net.SplitHostPort(addr)
	if host == "::1" || host == "127.0.0.1" {
		return net.Dial(network, addr)
	}
	use := self.UseProxy(addr, nil)
	if use {
		return self.SSH.Direct.Tr.Dial(network, addr)
	} else {
		value, has := self.Cfg.File.ForwardMap[addr]
		if has {
			addr = value
		}
	}
	return net.Dial(network, addr)
}
