package qdefine

// BrokerConfig 主服务配置
type BrokerConfig struct {
	Addr           string
	UId            string
	Pwd            string
	LogMode        string
	TimeOut        int
	Retry          int
	DetectedRoutes []string // 需要对外暴露的方法列表
}

type (
	// InitHandler 初始化回调
	InitHandler func(moduleName string)
	// ReqHandler 请求回调
	ReqHandler func(route string, ctx Context) (any, error)
	// NoticeHandler 通知回调
	NoticeHandler func(route string, ctx Context)
	// DetectedHandler 检测回调
	DetectedHandler func(fromModule, route string, ctx Context)
	// StateHandler 状态回调
	StateHandler func(state ECommState)
)

type (
	// SendRequestHandler 发送请求方法定义
	SendRequestHandler func(module, route string, params any) (Context, error)
)
