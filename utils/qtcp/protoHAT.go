package qtcp

import (
	"errors"
)

// hATPacket 头尾断帧协议
// 头尾断帧包 包尾必须不为空
type hATPacket struct {
	//Head 包头定义
	Head []byte
	//TypeBytes 包类型定义
	TypeBytes []byte
	//Body 包内容
	Body []byte
	//Tail 包尾
	Tail []byte
	//Check 校验 跟在包尾后方
	CheckBytes []byte
}

// ToString 转化为字符串 不包括校验
func (pack *hATPacket) ToString() string {
	return string(pack.Head) + string(pack.Body) + string(pack.Tail)
}

// newHATPacket 创建新包
func newHATPacket(head, typeBytes, body, tail, checkBytes []byte) *hATPacket {
	return &hATPacket{
		Head:       head,
		TypeBytes:  typeBytes,
		Body:       body,
		Tail:       tail,
		CheckBytes: checkBytes,
	}
}

// Marshal 组包
func (pack *hATPacket) Marshal() []byte {
	b := pack.Head
	if pack.TypeBytes != nil {
		b = append(b, pack.TypeBytes...)
	}
	b = append(b, pack.Body...)
	b = append(b, pack.Tail...)
	b = append(b, pack.CheckBytes...)
	return b
}

// Split 拆包
func (pack *hATPacket) Split() (frameType, body []byte) {
	return nil, pack.Body
}

type hatProtocol struct {
	//Head  包头
	Head []byte

	//Tail  包尾
	Tail []byte

	//TypeLen  帧类型长度
	TypeLen int

	//CheckType 校验类型
	CheckType ECheckType

	//CheckLen  校验位长度
	CheckLen int
	//onCheckPacket 校验方法
	onCheckPacket CheckPacketCallBack
	//buff          []byte
}

func (protoc *hatProtocol) BuildFrame(typeBytes, content []byte) (Packet, error) {
	if len(typeBytes) != protoc.TypeLen {
		return nil, errors.New("typeBytes length is not matched")
	}
	zeroCheckBytes := make([]byte, protoc.CheckLen)
	pack := newHATPacket(protoc.Head, typeBytes, content, protoc.Tail, zeroCheckBytes)
	if protoc.onCheckPacket != nil {
		check, _ := protoc.onCheckPacket(pack)
		pack.CheckBytes = check
	}

	return pack, nil
}

func NewHatProtocol(head, tail []byte, typeLen int, checkType ECheckType) PackProtocol {
	p := &hatProtocol{
		Head:      head,
		Tail:      tail,
		CheckType: checkType,
		TypeLen:   typeLen,
		//buff:          make([]byte, 0),
	}
	switch checkType {
	case ECheckTypeCheckSum:
		p.CheckLen = 2
		p.onCheckPacket = p.check
	case ECheckTypeCRC16:
		p.CheckLen = 2
		p.onCheckPacket = p.check
	}
	return p

}

// GetFrame  断帧
func (protoc *hatProtocol) GetFrame(buff *[]byte, recChan chan<- Packet) error {
Start:
	buf := *buff
	headIndex := find(buf, protoc.Head)

	if headIndex < 0 { //没找到头 清空缓存继续等待
		*buff = []byte{}
		return nil
	}

	tailIndex := find(buf, protoc.Tail)
	if tailIndex < 0 { //没找到尾 说明是中间段 不用做任何事
		return nil
	}
	if tailIndex <= headIndex { //尾大于头 提示故障并丢掉头之前的数据
		*buff = buf[headIndex:]
		return errors.New("GetFrame error  tail is before head")
	}
	if tailIndex+len(protoc.Tail)+protoc.CheckLen > len(buf) { //校验数据还没收完整 继续收
		return nil
	}
	headLen := len(protoc.Head)
	tailLen := len(protoc.Tail)
	//获取完整包
	var head, typeBytes, body, tail, checkBytes []byte
	i := headIndex
	j := headIndex + headLen
	head = buf[i:j]
	if protoc.TypeLen > 0 {
		i = j
		j += protoc.TypeLen
		typeBytes = buf[i:j]
	}

	i = j
	j = tailIndex
	body = buf[i:j]
	i = j
	j = j + tailLen
	tail = buf[i:j]
	if protoc.CheckLen > 0 {
		i = j
		j = j + protoc.CheckLen
		checkBytes = buf[i:j]
	}

	pack := newHATPacket(head, typeBytes, body, tail, checkBytes)

	//清除缓冲区之前的数据
	*buff = buf[j:]

	if protoc.onCheckPacket == nil {
		recChan <- pack
	} else if _, b := protoc.onCheckPacket(pack); b {
		recChan <- pack
	} else { //校验失败
		return errors.New("packet check failed")
	}
	if len(*buff) > 0 {
		goto Start
	}
	return nil

}

// 查找 从左到右适合找头 从右到左适合找尾
func find(target []byte, refer []byte) int {
	targetLen := len(target)
	referLen := len(refer)
	i := 0

	for {
		if i+referLen > targetLen {
			return -1
		}
		matched := false
		for j, v := range refer {

			if target[i+j] != v {
				matched = false
				break
			}
			matched = true
		}
		if matched {
			return i
		}
		i++
		if i >= targetLen {
			return -1
		}

	}
}

func (protoc *hatProtocol) check(pack Packet) ([]byte, bool) {
	return onCheck(protoc.CheckType, protoc.CheckLen, pack)
}

//func (protoc *hatProtocol) Crc16(pack Packet) ([]byte, bool) {
//	return onCheck(protoc.CheckType, protoc.CheckLen, pack)
//}
