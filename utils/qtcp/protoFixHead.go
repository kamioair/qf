package qtcp

import (
	"errors"
)

// fHPacket 固定头包 包结构为 特征头-包类型-包长度-正文-校验顺序不能变
type fHPacket struct {
	//Head 特征头
	Head []byte
	//包类型
	TypeBytes []byte

	//LenBytes 包长
	LenBytes []byte

	//Body 包内容
	Body []byte

	//Check 校验
	CheckBytes []byte
}

// NewFixHeadPacket 创建新固定头包
func newFHPacket(head, typeBytes, lenBytes, body, checkBytes []byte) *fHPacket {
	return &fHPacket{
		Head:       head,
		TypeBytes:  typeBytes,
		LenBytes:   lenBytes,
		Body:       body,
		CheckBytes: checkBytes,
	}
}

// Marshal 组包
func (pack *fHPacket) Marshal() []byte {
	b := make([]byte, 0, 100)
	b = append(b, pack.Head...)
	b = append(b, pack.TypeBytes...)
	b = append(b, pack.LenBytes...)
	b = append(b, pack.Body...)
	b = append(b, pack.CheckBytes...)
	return b
}

// Split 拆包
func (pack *fHPacket) Split() (frameType, body []byte) {
	return pack.CheckBytes, pack.Body
}

// fHProtocol 固定包头协议  固定头包 包结构为 特征头-包类型-包长度-正文-校验顺序不能变
type fHProtocol struct {

	//Head 特征头
	Head []byte

	//TypeLen 包类型长度 字节
	TypeLen int

	//LenSize 包长位数 仅支持8 16 32 64
	LenSize int

	//CheckLen  校验长度 字节
	CheckLen int

	onCheckPacket CheckPacketCallBack
	checkType     ECheckType
	//包长是高位在前还是低位在前
	IsBigEndian bool
	//最小包长
	MinLength int
}

// NewFHProtocol 新建固定包头协议
// head 特征头 typeLen 包类型长度 字节 lenSize 包长位数 仅支持8 16 32   CheckType 校验方法
func NewFHProtocol(head []byte, typeLen, lenSize int, bigEndian bool, checkType ECheckType) PackProtocol {
	p := &fHProtocol{
		Head:        head,
		TypeLen:     typeLen,
		LenSize:     lenSize,
		checkType:   checkType,
		IsBigEndian: bigEndian,
	}
	p.MinLength = len(head) + typeLen + lenSize/8

	switch checkType {
	case ECheckTypeCheckSum:
		p.CheckLen = 2
		p.onCheckPacket = p.check
		p.MinLength += 2
	case ECheckTypeCRC16:
		p.CheckLen = 2
		p.onCheckPacket = p.check
		p.MinLength += 2
	}
	return p
}

// GetFrame  断帧
func (protoc *fHProtocol) GetFrame(buff *[]byte, recChan chan<- Packet) error {
	if len(*buff) < protoc.MinLength {
		return nil //长度不够 继续等待
	}
	headIndex := 0
	i := 0
	var head, typeBs, lenBs, body, checkBs []byte
	buf := *buff
	if protoc.Head != nil { //包头

		headIndex = find(buf, protoc.Head)

		if headIndex < 0 { //没找到头 清空缓存继续等待
			*buff = []byte{}
			return nil
		}
		i = headIndex
		head = buf[i : i+len(protoc.Head)]
		i += len(protoc.Head)
	}
	if protoc.TypeLen > 0 { //包类型
		typeBs = buf[i : i+protoc.TypeLen]
		i += protoc.TypeLen
	}
	lenBs = buf[i : i+protoc.LenSize/8]

	bodyLen, err := Bytes.BtoI(lenBs, protoc.IsBigEndian)
	if err != nil {
		*buff = make([]byte, 0) //清空数据
		return err
	}
	//for i, v := range lenBs {
	//	if i < protoc.LenSize/8-1 {
	//		bodyLen += bodyLen<<8 + int(v)
	//	} else {
	//		bodyLen += int(v)
	//	}
	//}
	if len(buf) < bodyLen+i+protoc.CheckLen { //长度不够 继续等待
		return nil
	}
	i += protoc.LenSize / 8
	body = buf[i : i+bodyLen]
	i += bodyLen

	//checkBs = buf[i : i+protoc.CheckLen]
	if protoc.CheckLen > 0 {
		checkBs = buf[i : i+protoc.CheckLen]
		i += protoc.CheckLen
	}
	//获取完整包
	pack := newFHPacket(head, typeBs, lenBs, body, checkBs)
	//清除缓冲区之前的数据
	*buff = buf[i:]
	if protoc.onCheckPacket == nil || protoc.CheckLen == 0 { //校验一下 确认包没问题
		recChan <- pack
		return nil
	}
	if _, b := protoc.onCheckPacket(pack); b {
		recChan <- pack
		return nil
	} else { //校验失败
		return errors.New("packet check failed")
	}
}

// BuildFrame 从内容创建帧
func (protoc *fHProtocol) BuildFrame(typeBytes, content []byte) (Packet, error) {
	if len(typeBytes) != protoc.TypeLen {
		return nil, errors.New("typeBytes length is not matched")
	}
	lenBytes, e := Bytes.ItoB(len(content), protoc.IsBigEndian, protoc.LenSize)
	if e != nil {
		return nil, e
	}

	zeroCheckBytes := make([]byte, protoc.CheckLen)
	pack := newFHPacket(protoc.Head, typeBytes, lenBytes, content, zeroCheckBytes)
	if protoc.onCheckPacket != nil {
		check, _ := protoc.onCheckPacket(pack)
		pack.CheckBytes = check
	}

	return pack, nil
}

//func (protoc *fHProtocol) CheckSum(pack Packet) ([]byte, bool) {
//	return onCheck(protoc.checkType, protoc.CheckLen, pack)
//}

func (protoc *fHProtocol) check(pack Packet) ([]byte, bool) {
	return onCheck(protoc.checkType, protoc.CheckLen, pack)
}
