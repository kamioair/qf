package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/utils/qconvert"
	"github.com/kamioair/utils/qio"
	"github.com/kamioair/utils/qlauncher"
	easyCon "github.com/qiu-tec/easy-con.golang"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type IModule interface {
	Run()
	Stop()
}

// NewModule 创建模块
func NewModule(name, desc, version string, service IService, config IConfig) IModule {
	if service == nil {
		panic(errors.New("service cannot be nil"))
	}

	// 加载配置
	loadConfig(name, desc, version, config)

	// 创建基础模块
	m := &module{
		service: service,
		config:  config,
	}
	return m
}

type module struct {
	service IService
	reg     *Reg
	config  IConfig
	adapter easyCon.IAdapter
}

// Run 运行模块
func (m *module) Run() {
	qlauncher.Run(m.start, m.stop, false)
}

// Stop 停止模块
func (m *module) Stop() {
	qlauncher.Exit()
}

func (m *module) start() {
	defer m.errRecover(func(err string) {
		fmt.Println("")
		fmt.Println(err)
		fmt.Println("-------------------------------------")
	})

	cfg := m.config.getBaseConfig()
	fmt.Println("-------------------------------------")
	fmt.Println(" Module:", cfg.module)
	fmt.Println(" Desc:", cfg.desc)
	fmt.Println(" ModuleVersion:", cfg.version)
	fmt.Println(" FrameVersion:", Version)
	fmt.Println("-------------------------------------")

	m.reg = &Reg{}
	m.service.Reg(m.reg)
	m.service.setConfig(m.config.getBaseConfig())
	m.service.setWriteLog(m.writeLog)

	fmt.Printf("Connecting Broker... (Addr: %s)\n\n", cfg.Broker.Addr)
	// 创建easyCon客户端
	clientId := cfg.module
	setting := easyCon.NewSetting(clientId, cfg.Broker.Addr, m.onReq, m.onState)
	if m.reg.OnNotice != nil {
		setting.OnNotice = m.reg.OnNotice
	}
	if m.reg.OnRetainNotice != nil {
		setting.OnRetainNotice = m.reg.OnRetainNotice
	}
	setting.UID = cfg.Broker.UId
	setting.PWD = cfg.Broker.Pwd
	setting.TimeOut = time.Duration(cfg.Broker.TimeOut) * time.Millisecond
	setting.ReTry = cfg.Broker.Retry
	setting.LogMode = easyCon.ELogMode(cfg.Broker.LogMode)
	setting.PreFix = cfg.Broker.Prefix
	setting.OnExiting = m.onExiting
	setting.OnGetVersion = m.onGetVersion
	if m.reg.OnLog != nil {
		setting.OnLog = m.reg.OnLog
	}
	m.adapter = easyCon.NewMqttAdapter(setting)
	// 等待确保连接成功
	time.Sleep(time.Millisecond * 100)

	// 调用业务的初始化
	m.service.setAdapter(m.adapter)

	if m.reg.OnInit != nil {
		m.reg.OnInit()
	}

	// 保存配置文件
	saveConfigFile()

	if setting.LogMode == easyCon.ELogModeConsole {
		fmt.Println("")
	}
	fmt.Println("Start OK")
	fmt.Println("-------------------------------------")
}

func (m *module) stop() {
	// 调用业务的退出
	if m.reg.OnStop != nil {
		m.reg.OnStop()
	}
	m.service.stop()
	// 退出客户端
	if m.adapter != nil {
		m.adapter.Stop()
	}
	fmt.Println("Module stop ok")
}

func (m *module) onExiting() {
	qlauncher.Exit()
}

func (m *module) onReq(pack easyCon.PackReq) (code easyCon.EResp, resp any) {
	defer m.errRecover(func(err string) {
		code = easyCon.ERespError
		resp = errors.New(err)
		// 记录日志
		str, _ := json.Marshal(pack)
		m.writeLog("Error", fmt.Sprintf("OnReq InParam=%s", str), err)
	})
	switch pack.Route {
	case "Exit":
		m.Stop()
		return easyCon.ERespSuccess, nil
	case "Version":
		ver := map[string]string{}
		cfg := m.config.getBaseConfig()
		ver["Module"] = cfg.module
		ver["Desc"] = cfg.desc
		ver["ModuleVersion"] = cfg.version
		ver["FrameVersion"] = Version
		return easyCon.ERespSuccess, ver
	}
	if m.reg.OnReq != nil {
		return m.reg.OnReq(pack)
	}
	return easyCon.ERespRouteNotFind, "Route Not Matched"
}

func (m *module) onState(status easyCon.EStatus) {
	if m.reg.OnStatusChanged != nil {
		m.reg.OnStatusChanged(status)
	}
}

// @Description: Panic的异常收集
func (m *module) errRecover(after func(err string)) {
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
					errStr += m.formatStack(lines[i+2], lines[i+3])
				}
				if i+5 < len(lines) {
					errStr += m.formatStack(lines[i+4], lines[i+5])
				}
			}
			log += fmt.Sprintf(" %s\n", lines[i])
		}

		// 执行外部方法
		if after != nil {
			after(log)
		}
	}
}

func (m *module) formatStack(name string, row string) string {
	sp := strings.Split(strings.Replace(row, "\t", "", -1), "+")
	funcName := filepath.Base(name)
	matches := regexp.MustCompile(`\((.*?)\)`).FindAllStringSubmatch(funcName, -1)
	if matches != nil && len(matches) > 0 && len(matches[len(matches)-1]) > 0 {
		funcName = strings.Replace(funcName, matches[len(matches)-1][0], "(...)", 1)
	}
	return fmt.Sprintf("   %s\n      %s\n", funcName, sp[0])
}

func (m *module) writeLog(level string, content string, err string) {
	baseCfg := m.config.getBaseConfig()
	now := time.Now()
	temp := "{Time} [{Level}] {Content} {Error}"
	log := strings.Replace(temp, "{Time}", qconvert.Time.ToString(now, "yyyy-MM-dd HH:mm:ss"), 1)
	log = strings.Replace(log, "{Level}", level, 1)
	log = strings.Replace(log, "{Content}", content, 1)
	log = strings.Replace(log, "{Error}", err, 1)
	ym := qconvert.Time.ToString(now, "yyyy-MM")
	day := qconvert.Time.ToString(now, "dd")
	logFile := fmt.Sprintf("%s/%s/%s_%s_%s.log", "./log", ym, day, baseCfg.module, "Error")
	logFile = qio.GetFullPath(logFile)
	_ = qio.WriteString(logFile, log, true)
}

func (m *module) onGetVersion() []string {
	return []string{fmt.Sprintln("qf:", Version), fmt.Sprintln("module:", m.config.getBaseConfig().version)}
}
