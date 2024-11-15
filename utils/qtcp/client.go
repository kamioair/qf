package qtcp

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type client struct {
	svrAddr         *net.TCPAddr
	relinkWaitTime  time.Duration
	keepAlivePeriod time.Duration
	relinkChan      chan struct{}
	connection      Connection
	baseInfo
	isRunning int32
	//isLinked int32
	linkFailed bool
}

func (client *client) Send(pack Packet, timeout time.Duration) error {
	if client.connection == nil {
		return ErrConnClosed
	}
	return client.connection.Send(pack, timeout)
}

func (client *client) GetId() int64 {
	if client.connection == nil {
		return -1
	}
	return client.connection.GetId()
}

func NewClient(svrAddr string, buffLength int, protocol PackProtocol, callback ConnCallback, relinkWaitTime, keepAlivePeriod time.Duration) Client {
	if protocol == nil {
		panic("tcp.NewClient: protoc can not be nil")
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", svrAddr)
	Check(err)
	return &client{
		svrAddr:         tcpAddr,
		relinkChan:      make(chan struct{}, 1),
		relinkWaitTime:  relinkWaitTime,
		keepAlivePeriod: keepAlivePeriod,
		baseInfo: baseInfo{
			buffLength: buffLength,
			//waitGroup:  &sync.WaitGroup{},
			callback: callback,
			protocol: protocol,
			//closeOnce:  &sync.Once{},
			//closeChan: make(chan struct{}),
		},
	}
}
func (client *client) IsClosed() bool {
	if client.connection == nil {
		return false
	}
	return client.connection.IsClosed()
}
func (client *client) Start() {

	if atomic.LoadInt32(&client.isRunning) == 1 {
		client.callback.OnErrored(errors.New("client is already running "), nil)
		return
	}
	atomic.StoreInt32(&client.isRunning, 1)
	client.waitGroup = &sync.WaitGroup{}
	client.closeChan = make(chan struct{})
	client.waitGroup.Add(1)
	go client.handleLoop()
	client.CallRelink()
}

// 自动重连
func (client *client) handleLoop() {
	defer client.waitGroup.Done()
	//client.link()
	for true {
		select {
		case <-client.closeChan:
			return
		case <-client.relinkChan:
			//	if client.linkFailed { //如果已经连接失败了，则等待一段时间
			//		time.Sleep(client.relinkWaitTime)
			//	}
			client.link()
		}
	}
}

func (client *client) link() {
	//if atomic.LoadInt32(&client.linked) == 1 {
	//	return
	//}
	d := net.Dialer{Timeout: time.Second}

	//conn, err := net.DialTCP("tcp", nil, client.svrAddr)
	rawConn, err := d.Dial("tcp", client.svrAddr.String())
	if err != nil {
		if !client.linkFailed {
			client.callback.OnErrored(err, nil)
			client.linkFailed = true
		}
		go func() {
			time.Sleep(client.relinkWaitTime) //已经失败了 等待一个时间再发送重连请求
			client.CallRelink()
		}()
		return
	}
	conn := rawConn.(*net.TCPConn)
	b := client.baseInfo
	//b.closeOnce = &sync.Once{}
	b.waitGroup = &sync.WaitGroup{}
	b.callback = client
	c := newConn(1, conn, b, client.keepAlivePeriod)
	c.Start()
	client.connection = c
	client.linkFailed = false

}

func (client *client) Stop() {
	if atomic.LoadInt32(&client.isRunning) == 0 {
		client.callback.OnErrored(errors.New("client is already stopped "), nil)
		return
	}
	//client.closeOnce.Do(func() {
	close(client.closeChan)
	client.waitGroup.Wait()
	//})

	atomic.StoreInt32(&client.isRunning, 0)
}

func (client *client) OnLinked(c Connection) {
	client.callback.OnLinked(c)
}

func (client *client) OnReceived(c Connection, packet Packet) {
	client.callback.OnReceived(c, packet)
}

func (client *client) OnClosed(c Connection) {
	client.callback.OnClosed(c)
	client.CallRelink()
}

func (client *client) OnErrored(e error, c Connection) {
	client.callback.OnErrored(e, c)
}

func (client *client) CallRelink() {
	select {
	case client.relinkChan <- struct{}{}:
		fmt.Println("relink signal sent")
	default:
		//正在relink中，不需要relink
		fmt.Println("relinkChan is full doesn't need to call")
	}
}
