package qtcp

import (
	"sync"
	"time"
)

type baseInfo struct {
	buffLength int
	//同步等待组
	waitGroup *sync.WaitGroup
	//委托
	callback ConnCallback
	//协议
	protocol PackProtocol

	//关闭
	closeChan chan struct{}
}

type Client interface {
	Connection
	Stop()
}
type Server interface {
	Start()
	Stop()
	//CloseConnection(id int) (bool, error)
	//Send(id int, pack Packet) error
}
type Connection interface {
	Start()
	Send(pack Packet, timeout time.Duration) error
	GetId() int64
	IsClosed() bool
}

// PackProtocol 封包协议
type PackProtocol interface {
	//GetFrame 组包 如果成功 推一个包到接收缓冲区
	GetFrame(b *[]byte, recChan chan<- Packet) error
	BuildFrame(typeBytes, content []byte) (Packet, error)
	//OnReceived 进程 收到一个完整包后触发
	//OnReceived(conn Connection, packet Packet)
}

// Packet 数据包
type Packet interface {
	// Marshal 编码方法 数据包必需实现将其内容格式化为byte数组的编码方法
	Marshal() []byte

	Split() (frameType, body []byte)
}

// ConnCallback 建立连接时的委托定义
type ConnCallback interface {
	OnLinked(c Connection)
	OnReceived(c Connection, packet Packet)
	OnClosed(c Connection)
	OnErrored(e error, c Connection)
}

type CheckPacketCallBack func(pack Packet) ([]byte, bool)
