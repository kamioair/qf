package qtcp

import (
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type server struct {
	port            int
	acceptTimeout   time.Duration
	keepAlivePeriod time.Duration
	baseInfo
	nextId     int64
	AcceptChan chan struct{}
}

func NewServer(port int, acceptTimeout, keepAlivePeriod time.Duration, buffLength int, callback ConnCallback, protocol PackProtocol) Server {
	if protocol == nil {
		panic("tcp.NewClient: protocol can not be nil")
	}
	return &server{
		port:            port,
		acceptTimeout:   acceptTimeout,
		keepAlivePeriod: keepAlivePeriod,
		AcceptChan:      make(chan struct{}),
		baseInfo: baseInfo{
			buffLength: buffLength,
			waitGroup:  &sync.WaitGroup{},
			callback:   callback,
			protocol:   protocol,
			//closeOnce:  &sync.Once{},
			closeChan: make(chan struct{}),
		},
		nextId: 0,
	}
}
func (server *server) GetNextId() int64 {
	return atomic.AddInt64(&server.nextId, 1)
}
func (server *server) Start() {

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(server.port))
	if err != nil {
		server.callback.OnErrored(err, nil)
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		server.callback.OnErrored(err, nil)
	}
	server.waitGroup.Add(1)
	defer func() {
		_ = listener.Close()
		server.waitGroup.Done()
	}()
	server.waitGroup.Add(1)
	go server.accept(listener)
	for {
		select {
		case <-server.closeChan:
			return
		case <-server.AcceptChan:
			server.waitGroup.Add(1)
			go server.accept(listener)
		}

	}
}

func (server *server) accept(listener *net.TCPListener) {
	defer func() {
		server.waitGroup.Done()
		server.AcceptChan <- struct{}{}
	}()
	if server.acceptTimeout != 0 {
		deadline := time.Now().Add(server.acceptTimeout)
		_ = listener.SetDeadline(deadline)
	}
	conn, err := listener.AcceptTCP()

	if err != nil {
		return
	}
	b := server.baseInfo
	//b.closeOnce = &sync.Once{}
	b.waitGroup = &sync.WaitGroup{}
	c := newConn(server.GetNextId(), conn, b, server.keepAlivePeriod)
	c.Start()
	//server.callback.OnLinked(c)
	//server.waitGroup.Done()
}

func (server *server) Stop() {
	//server.closeOnce.Do(func() {
	close(server.closeChan)
	server.waitGroup.Wait()
	//})
}
