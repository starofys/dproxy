package dproxy

import (
	"io"
	"net"
	"time"
)

func (self *Server) ServerTunnel() {
	for _, item := range self.Cfg.File.TunnelMap {
		if len(item) != 3 {
			L.Printf("error config %s", item)
			continue
		}
		go self.initServerTunnel(item[0], item[1], item[2])
	}
}

func (self *Server) initServerTunnel(local, remote, name string) {

	ln, err := net.Listen("tcp", local)

	if err != nil {
		L.Printf("listen fail %s %s => %s %s", name, local, remote, err)
		return
	}

	L.Printf("new Tunnel %s %s => %s\n", name, local, remote)

	for id := 0; ; id++ {
		conn, err := ln.Accept()
		if err != nil {
			L.Printf("%d: %s\n", id, err)
			continue
		}
		L.Printf("%d: new %s %s %s\n", id, name, local, remote)

		if tcpConn := conn.(*net.TCPConn); tcpConn != nil {
			// L.Printf("%d: setup keepalive for TCP connection\n", id)
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}

		go func(myid int, conn net.Conn) {
			defer conn.Close()
			c, err := self.SSH.Direct.Tr.Dial("tcp", remote)
			if err != nil {
				L.Printf("%d: %s\n", myid, err)
				return
			}
			L.Printf("%d: new %s <-> %s\n", myid, local, remote)
			defer c.Close()
			wait1 := make(chan int)
			wait2 := make(chan int)
			go func() {
				n, err := io.Copy(c, conn)
				if err != nil {
					L.Printf("%d: %s\n", myid, err)
				}
				L.Printf("%d: %s -> %s %d bytes\n", myid, remote, local, n)
				close(wait1)
			}()
			go func() {
				n, err := io.Copy(conn, c)
				if err != nil {
					L.Printf("%d: %s\n", myid, err)
				}
				L.Printf("%d: %s -> %s %d bytes\n", myid, local, remote, n)
				close(wait2)
			}()
			<-wait1
			<-wait2
			L.Printf("%d: connection closed\n", myid)
		}(id, conn)
	}

}
