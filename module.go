package qf

import (
	"encoding/json"
	"errors"
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
)

// baseModule 基础模块，包含所有模块类型的公共实现
type baseModule struct {
	service IService
	reg     *Reg
	adapter easyCon.IAdapter
}

// newBaseModule 创建基础模块
func newBaseModule(service IService) *baseModule {
	if service == nil {
		panic(errors.New("service cannot be nil"))
	}

	bm := &baseModule{
		service: service,
		reg:     &Reg{},
	}
	bm.service.Reg(bm.reg)

	return bm
}

// getService 获取服务接口
func (bm *baseModule) getService() IService {
	return bm.service
}

// getReg 获取注册对象
func (bm *baseModule) getReg() *Reg {
	return bm.reg
}

// getAdapter 获取适配器
func (bm *baseModule) getAdapter() easyCon.IAdapter {
	return bm.adapter
}

// printModuleInfo 打印模块启动信息
func (bm *baseModule) printModuleInfo() {
	cfg := bm.service.config().getBase()
	fmt.Println("-------------------------------------")
	fmt.Println(" Module:", cfg.module)
	fmt.Println(" Desc:", cfg.desc)
	fmt.Println(" ModuleVersion:", cfg.version)
	fmt.Println(" FrameVersion:", Version)
	fmt.Println("-------------------------------------")
}

// saveConfig 保存配置文件
func (bm *baseModule) saveConfig() {
	saveConfigFile(bm.service.config())
}

// buildAdapterCallBack 构建 easyCon 适配器回调
func (bm *baseModule) buildAdapterCallBack(
	onStatusChanged easyCon.StatusChangedHandler,
	onReq easyCon.ReqHandler,
	onExiting func(),
	onGetVersion func() []string,
) easyCon.AdapterCallBack {
	callback := easyCon.AdapterCallBack{
		OnStatusChanged: onStatusChanged,
		OnReqRec:        onReq,
		OnRespRec:       nil,
		OnExiting:       onExiting,
		OnGetVersion:    onGetVersion,
	}
	if bm.reg.OnNotice != nil {
		callback.OnNoticeRec = bm.reg.OnNotice
	}
	if bm.reg.OnRetainNotice != nil {
		callback.OnRetainNoticeRec = bm.reg.OnRetainNotice
	}
	if bm.reg.OnLog != nil {
		callback.OnLogRec = bm.reg.OnLog
	}
	return callback
}

func (bm *baseModule) callOnState(status easyCon.EStatus) {
	fmt.Printf("Link state = [%s]\n", status)
	if bm.reg != nil && bm.reg.OnStatusChanged != nil {
		go bm.reg.OnStatusChanged(status)
	}
}

// callOnInit 调用业务初始化回调
func (bm *baseModule) callOnInit() {
	bm.service.setEnv(bm.reg, bm.adapter)
	if bm.reg.OnInit != nil {
		bm.reg.OnInit()
	}
}

// callOnStop 调用业务停止回调
func (bm *baseModule) callOnStop() {
	if bm.reg.OnStop != nil {
		bm.reg.OnStop()
	}
}

// decryptBrokerConfig 解密 Broker 配置
func (bm *baseModule) decryptBrokerConfig() (addr, uid, pwd string) {
	cfg := bm.service.config().getBase()

	addr = cfg.Broker.Addr
	uid = cfg.Broker.UId
	pwd = cfg.Broker.Pwd

	if cfg.crypto != nil {
		nAddr, e := cfg.crypto.Decrypt(addr)
		if e == nil {
			addr = nAddr
		}
		nUid, e := cfg.crypto.Decrypt(uid)
		if e == nil {
			uid = nUid
		}
		nPwd, e := cfg.crypto.Decrypt(pwd)
		if e == nil {
			pwd = nPwd
		}
	}

	return
}

// handleReq 通用请求处理
func (bm *baseModule) handleReq(pack easyCon.PackReq, onStop func()) (code easyCon.EResp, resp []byte) {
	cfg := bm.service.config().getBase()

	defer errRecover(func(err string) {
		code = easyCon.ERespError
		resp = []byte(err)
	}, cfg.module, pack.Route, pack.Content)

	switch pack.Route {
	case "Exit":
		if onStop != nil {
			onStop()
		}
		return easyCon.ERespSuccess, nil
	case "Version":
		ver := map[string]string{}
		ver["Module"] = cfg.module
		ver["Desc"] = cfg.desc
		ver["ModuleVersion"] = cfg.version
		ver["FrameVersion"] = Version
		j, _ := json.Marshal(ver)
		return easyCon.ERespSuccess, j
	}

	if bm.reg.OnReq != nil {
		code, resp = bm.reg.OnReq(pack)
		if code != easyCon.ERespSuccess {
			// 记录日志
			str, _ := json.Marshal(pack.Content)
			errStr := string(resp)
			writeLog(cfg.module, "Error", fmt.Sprintf("[OnReq From %s.%s] InParam=%s", pack.From, pack.Route, str), formatRespError(code, errStr))
		}
		return code, resp
	}
	return easyCon.ERespRouteNotFind, []byte("Route Not Matched")
}

// getVersion 获取版本信息
func (bm *baseModule) getVersion() []string {
	cfg := bm.service.config().getBase()
	return []string{fmt.Sprintln("qf:", Version), fmt.Sprintln("module:", cfg.version)}
}

// stopAdapter 停止适配器
func (bm *baseModule) stopAdapter() {
	if bm.adapter != nil {
		bm.adapter.Stop()
	}
}
