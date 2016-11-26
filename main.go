package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"
)

var tunnel *Tunnel

type Tunnel struct {
	remoteAddr string
	tlsCfg     *tls.Config
	connCount  int32
}

func NewTunnel(remoteAddr string, strictSecure bool) *Tunnel {
	return &Tunnel{
		remoteAddr: remoteAddr,
		tlsCfg: &tls.Config{
			ClientSessionCache: tls.NewLRUClientSessionCache(256), // use sessoin ticket to speed up tls handshake
			InsecureSkipVerify: !strictSecure,
		},
	}
}

func (t *Tunnel) pipe(dst, src net.Conn, c chan int64) {
	n, err := io.Copy(dst, src)
	if err != nil {
		log.Print(err)
	}
	c <- n
}

func (t *Tunnel) HandleConn(conn *net.TCPConn) {
	log.Println("handle a new connection, conn:%d", atomic.LoadInt32(&t.connCount))
	start := time.Now()
	atomic.AddInt32(&t.connCount, 1)
	defer atomic.AddInt32(&t.connCount, -1)
	rConn, err := tls.Dial("tcp", t.remoteAddr, t.tlsCfg)
	if err != nil {
		log.Printf("connect to remote [%s] failed:%v", t.remoteAddr, err)
		return
	}
	readChan := make(chan int64)
	writeChan := make(chan int64)
	go t.pipe(rConn, conn, writeChan)
	go t.pipe(conn, rConn, readChan)

	var writeBytes int64
	var readBytes int64
	for i := 0; i < 2; i++ {
		select {
		case writeBytes = <-writeChan:
			conn.CloseRead()
		case readBytes = <-readChan:
			conn.CloseWrite()
		}
	}
	// TODO
	rConn.Close()

	log.Printf("conn:%d, read:%d, write:%d, cost:%v",
		atomic.LoadInt32(&t.connCount), readBytes, writeBytes, time.Now().Sub(start))
}

func main() {
	var (
		localAddr    string
		remoteAddr   string
		strictSecure bool
	)
	flag.StringVar(&localAddr, "l", "127.0.0.1:21126", "local listen address")
	flag.StringVar(&remoteAddr, "r", "", "remote TLS address")
	flag.BoolVar(&strictSecure, "s", true, "strict secure do NOT skip insecure certificate verify")
	flag.Parse()
	laddr, err := net.ResolveTCPAddr("tcp", localAddr)
	if err != nil {
		log.Printf("resolve local address failed:%v", err)
		os.Exit(1)
	}
	if remoteAddr == "" {
		log.Println("remote address is empty")
		os.Exit(1)
	}

	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		log.Printf("listen failed:%v", err)
		os.Exit(1)
	}
	tunnel = NewTunnel(remoteAddr, strictSecure)

	log.Println("start serving at [%s]", localAddr)
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Printf("accept connection error:%v", err)
			continue
		}
		go tunnel.HandleConn(conn)
	}
}
