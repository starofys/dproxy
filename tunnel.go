package dproxy

import (
	"io"
	"net"
	"time"
)

func (self *Server) ServerTunnel() {
	for _, item := range self.Cfg.File.TunnelMap {
		go self.initServerTunnel(&item)
	}
}

func (self *Server) initServerTunnel(item *TunnelItem) {

	ln, err := net.Listen("tcp", item.Local)

	if err != nil {
		L.Printf("listen fail %s %s => %s %s", item.Name, item.Local, item.Target, err)
		return
	}

	L.Printf("new Tunnel %s %s => %s\n", item.Name, item.Local, item.Target)

	for id := 0; ; id++ {
		conn, err := ln.Accept()
		if err != nil {
			L.Printf("%d: %s\n", id, err)
			continue
		}
		L.Printf("%d: new %s %s %s\n", id, item.Name, item.Local, item.Target)

		if tcpConn := conn.(*net.TCPConn); tcpConn != nil {
			// L.Printf("%d: setup keepalive for TCP connection\n", id)
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}

		go func(myid int, conn net.Conn) {
			defer conn.Close()
			c, err := self.SSH.Direct.Tr.Dial("tcp", item.Target)
			if err != nil {
				L.Printf("%d: %s\n", myid, err)
				return
			}
			L.Printf("%d: new %s <-> %s\n", myid, item.Local, item.Target)
			defer c.Close()
			wait1 := make(chan int)
			wait2 := make(chan int)
			go func() {
				n, err := io.Copy(c, conn)
				if err != nil {
					L.Printf("%d: %s\n", myid, err)
				}
				L.Printf("%d: %s -> %s %d bytes\n", myid, item.Target, item.Local, n)
				close(wait1)
			}()
			go func() {
				n, err := io.Copy(conn, c)
				if err != nil {
					L.Printf("%d: %s\n", myid, err)
				}
				L.Printf("%d: %s -> %s %d bytes\n", myid, item.Local, item.Target, n)
				close(wait2)
			}()
			<-wait1
			<-wait2
			L.Printf("%d: connection closed\n", myid)
		}(id, conn)
	}

}
