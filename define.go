package qf

import (
	"encoding/json"
	"fmt"
	"github.com/kamioair/utils/qconvert"
	"github.com/kamioair/utils/qio"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// IModule 模块入口接口
type IModule interface {
	// Run 同步运行模块，执行后会等待直到程序退出
	Run()
	// RunAsync 异步运行模块，执行后不等待
	RunAsync()
	// Stop 停止模块
	Stop()
	// Name 获取模块名称
	Name() string
}

// IService 模块功能接口
type IService interface {
	Name() string                                                                          // 返回模块名称
	Reg(reg *Reg)                                                                          // 注册事件
	GetRegEvents() *Reg                                                                    // 返回注册事件
	Load(moduleName, moduleDesc, moduleVersion string, sectionName string, config IConfig) // 加载模块

	// 内部使用的方法
	config() IConfig
	setEnv(reg *Reg, adapter easyCon.IAdapter)
}

// IConfig 配置接口
type IConfig interface {
	setBase(moduleName, moduleDesc, moduleVersion string, sectionName string)
	getBase() *Config
}

// ICrypto 加解密接口
type ICrypto interface {
	// Decrypt 解密
	Decrypt(content string) (string, error)
}

// IContext 上下文
type IContext interface {
	Raw() string
	Bind(refStruct any) error
}

// Void 空值
type Void struct {
}

// Reg 事件绑定
type Reg struct {
	OnInit          func()
	OnStop          func()
	OnReq           func(pack easyCon.PackReq) (easyCon.EResp, []byte)
	OnNotice        func(notice easyCon.PackNotice)
	OnRetainNotice  func(notice easyCon.PackNotice)
	OnStatusChanged func(status easyCon.EStatus)
	OnLog           func(log easyCon.PackLog)
}

// OnReqFunc 请求方法定义
type OnReqFunc func(ctx IContext) (any, error)

// OnNoticeFunc 通知方法定义
type OnNoticeFunc func(ctx IContext)

// OnWriteDelegate 插件用委托
type OnWriteDelegate func([]byte) error
type OnReadDelegate func([]byte)

// Invoke 调用业务方法
func Invoke[T any](pack easyCon.PackReq, method T) (code easyCon.EResp, resp []byte) {
	defer errRecover(func(err string) {
		code = easyCon.ERespError
		resp = []byte(err)
	}, pack.To, pack.Route, pack.Content)

	// 验证
	v := reflect.ValueOf(method)
	if !v.IsValid() {
		return easyCon.ERespError, []byte("invalid method")
	}

	t := v.Type()
	// 检查是否是函数
	if t.Kind() != reflect.Func {
		return easyCon.ERespError, []byte(fmt.Sprintf("not a function: %T\n", method))
	}

	// 获取函数参数个数
	numIn := t.NumIn()
	if numIn > 1 {
		return easyCon.ERespError, []byte(fmt.Sprintf("method %T too many arguments, expect 0 or 1", method))
	}

	var args []reflect.Value
	var err error

	if numIn > 0 {
		paramType := t.In(0)
		if paramType.Kind() == reflect.String {
			args = []reflect.Value{reflect.ValueOf(string(pack.Content))}
		} else if paramType.Kind() == reflect.Slice {
			args = []reflect.Value{reflect.ValueOf(pack.Content)}
		} else {
			obj := reflect.New(paramType).Interface()
			err = json.Unmarshal(pack.Content, &obj)
			if err != nil {
				return easyCon.ERespBadReq, []byte(err.Error())
			}
			args = []reflect.Value{reflect.ValueOf(obj).Elem()}
		}
	}

	// 调用方法
	results := v.Call(args)

	// 处理返回
	if len(results) == 2 {
		code = results[0].Interface().(easyCon.EResp)
		if code != easyCon.ERespSuccess {
			if results[1].Interface() != nil {
				err = results[1].Interface().(error)
			}
			if err == nil {
				return code, []byte("")
			}
			return code, []byte(err.Error())
		}
		return code, nil
	} else if len(results) == 3 {
		code = results[1].Interface().(easyCon.EResp)
		if code != easyCon.ERespSuccess {
			if results[2].Interface() != nil {
				err = results[2].Interface().(error)
			}
			if err == nil {
				return code, []byte("")
			}
			return code, []byte(err.Error())
		}
		obj := results[0].Interface()
		if obj == nil {
			resp = []byte("")
		} else if s, ok := obj.(string); ok {
			// 如果是字符串，直接转换为 []byte
			resp = []byte(s)
		} else if b, ok := obj.([]byte); ok {
			// 如果已经是 []byte，直接使用
			resp = b
		} else {
			// 其他类型（结构体等）转换为 JSON 格式的 []byte
			var err error
			resp, err = json.Marshal(obj)
			if err != nil {
				return easyCon.ERespError, []byte(fmt.Sprintf("failed to marshal response: %v", err))
			}
		}
		return code, resp
	}
	return easyCon.ERespError, []byte("invalid return count, need any,code,error or code,error")
}

// @Description: Panic的异常收集
func errRecover(after func(err string), moduleName string, route string, inParam any) {
	if r := recover(); r != nil {
		// 获取异常
		var buf [4096]byte
		n := runtime.Stack(buf[:], false)
		stackInfo := string(buf[:n])

		// 输出异常
		log := ""
		log += fmt.Sprintf("%s\n", r)
		lines := strings.Split(stackInfo, "\n")
		for i := 0; i < len(lines); i++ {
			line := strings.Replace(lines[i], "\t", "", -1)
			if strings.HasPrefix(line, "panic") {
				errStr := ""
				if i+3 < len(lines) {
					errStr += formatStack(lines[i+2], lines[i+3])
				}
				if i+5 < len(lines) {
					errStr += formatStack(lines[i+4], lines[i+5])
				}
			}
			log += fmt.Sprintf(" %s\n", lines[i])
		}

		// 执行外部方法
		if after != nil {
			after(fmt.Sprintf("%v", r))
		}

		// 记录错误日志
		str, _ := json.Marshal(inParam)
		writeLog(moduleName, "Error", fmt.Sprintf("[%s] InParam=%s", route, str), log)
	}
}

func formatStack(name string, row string) string {
	sp := strings.Split(strings.Replace(row, "\t", "", -1), "+")
	funcName := filepath.Base(name)
	matches := regexp.MustCompile(`\((.*?)\)`).FindAllStringSubmatch(funcName, -1)
	if matches != nil && len(matches) > 0 && len(matches[len(matches)-1]) > 0 {
		funcName = strings.Replace(funcName, matches[len(matches)-1][0], "(...)", 1)
	}
	return fmt.Sprintf("   %s\n      %s\n", funcName, sp[0])
}

func writeLog(module string, level string, content string, err string) {
	now := time.Now()
	temp := "{Time} [{Level}] {Error} {Content}"
	log := strings.Replace(temp, "{Time}", qconvert.Time.ToString(now, "yyyy-MM-dd HH:mm:ss"), 1)
	log = strings.Replace(log, "{Level}", level, 1)
	log = strings.Replace(log, "{Error}", err, 1)
	log = strings.Replace(log, "{Content}", content, 1)
	ym := qconvert.Time.ToString(now, "yyyy-MM")
	day := qconvert.Time.ToString(now, "dd")
	logFile := fmt.Sprintf("%s/%s/%s_%s_%s.log", "./log", ym, day, module, level)
	logFile = qio.GetFullPath(logFile)
	_ = qio.WriteString(logFile, log+"\n", true)
}
