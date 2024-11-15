package qtcp

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Error type

var (
	ErrConnClosed = errors.New("the connection has been closed")
	//ErrWriteBlocked = errors.New("write packet was blocking")
)

// connection tcp连接
type connection struct {
	id int64
	//原始的tcp连接
	rawConn *net.TCPConn
	////发送chan
	//sendChan chan Packet
	//接收chan
	recChan chan Packet
	//关闭标志
	closedFlag int32
	//myCloseChan chan struct{}
	buf []byte
	//关闭单例 保证关闭仅被执行一次
	closeOnce *sync.Once

	baseInfo
}

func (conn *connection) GetId() int64 {
	return conn.id
}

// 构造
func newConn(id int64, c *net.TCPConn, baseInfo baseInfo, KeepAlivePeriod time.Duration) Connection {
	if KeepAlivePeriod > 0 {
		_ = c.SetKeepAlive(true)
		_ = c.SetKeepAlivePeriod(KeepAlivePeriod)
	}

	return &connection{
		id:      id,
		rawConn: c,
		//sendChan: make(chan Packet, 0),
		recChan: make(chan Packet, 0),
		//closedFlag:  0,
		//myCloseChan: make(chan struct{}),
		buf:       make([]byte, 0),
		baseInfo:  baseInfo,
		closeOnce: &sync.Once{},
	}

}

func (conn *connection) Start() {
	//启动读协程
	startGoroutine(conn.readLoop, conn.waitGroup)
	////启动写协程
	//startGoroutine(conn.writLoop, conn.waitGroup)

	//启动主控
	startGoroutine(conn.handle, conn.waitGroup)
	conn.callback.OnLinked(conn)
}

func (conn *connection) close() {
	conn.closeOnce.Do(func() {
		atomic.StoreInt32(&conn.closedFlag, 1)
		//close(conn.sendChan)
		close(conn.recChan)
		//close(conn.myCloseChan)
		_ = conn.rawConn.Close()
		conn.callback.OnClosed(conn)
	})
}

func (conn *connection) IsClosed() bool {
	return atomic.LoadInt32(&conn.closedFlag) == 1
}

// Send 发送包
func (conn *connection) Send(packet Packet, timeout time.Duration) (err error) {
	if conn.IsClosed() {
		err = ErrConnClosed
		return
	}
	//defer func() {
	//	if e := recover(); e != nil {
	//		e = ErrConnClosed
	//	}
	//}()
	if timeout > 0 {
		conn.rawConn.SetWriteDeadline(time.Now().Add(timeout))
	}
	if _, err = conn.rawConn.Write(packet.Marshal()); err != nil {
		conn.close()
		conn.callback.OnErrored(err, conn)
	}
	return
	//if conn.writeTimeout == 0 {
	//	select {
	//	case conn.sendChan <- packet:
	//		return nil
	//	default:
	//		return ErrWriteBlocked
	//	}
	//} else {
	//	select {
	//	case conn.sendChan <- packet:
	//		return nil
	//	case <-conn.closeChan:
	//		return ErrConnClosed
	//	case <-time.After(conn.writeTimeout):
	//		return ErrWriteTimeOut
	//	default:
	//		return ErrWriteBlocked
	//	}
	//}
}

//func (connection *connection) GetRawConn() *net.TCPConn {
//	return connection.rawConn
//}

// 启动一个grt并在外部封装wg操作以实现优雅退出
func startGoroutine(fn func(), wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		fn()
		wg.Done()
	}()
}

// 读数据方法 独立grt
func (conn *connection) readLoop() {
	defer conn.close()
	//设置切片作为缓冲区
	buf := make([]byte, conn.buffLength)
	for {
		select {
		//收到退出信号 则退出
		//case <-conn.myCloseChan:
		//	return
		case <-conn.closeChan:
			return
		default:
		}
		//conn.rawConn.SetReadDeadline(time.Now().Add(time.Second * 5))
		//读取数据到缓冲切片
		count, err := conn.rawConn.Read(buf)
		if err != nil {
			//连接已经断开 返回并执行退出
			//conn.callback.OnErrored(ErrConnClosed, conn)

			return
		}
		if count == 0 { //说明通信已经关闭 返回
			return
		}
		conn.buf = append(conn.buf, buf[:count]...)
		//装包并发送给接收管道（如果组包完成）
		e := conn.protocol.GetFrame(&conn.buf, conn.recChan)
		if e != nil {
			conn.callback.OnErrored(e, conn)
			return
		}

	}
}

////写数据方法 独立grt
//func (conn *connection) writLoop() {
//	//defer func() {
//	//	conn.waitGroup.Done()
//	//}()
//
//	for {
//		select {
//		//如果收到关闭信号则退出
//		case <-conn.closeChan:
//			return
//		//数据管道有数据需要发送
//		case p, b := <-conn.sendChan:
//			if !b { //Chan已经关闭
//				return
//			}
//			if _, err := conn.rawConn.Write(p.Marshal()); err != nil {
//				conn.callback.OnErrored(err, conn)
//				return
//			}
//		}
//	}
//}

// 主控方法 处理包
func (conn *connection) handle() {
	defer conn.close()
	for {
		select {
		case <-conn.closeChan:

			return
		case packet, b := <-conn.recChan:
			if !b { //recChan已经关闭
				return
			}
			conn.callback.OnReceived(conn, packet)
		}
	}
}
