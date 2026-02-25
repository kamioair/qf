package qf

import (
	"errors"
	"fmt"
	easyCon "github.com/qiu-tec/easy-con.golang"
)

type Service struct {
	adapterCon       easyCon.IAdapter
	cfg              IConfig
	requestFunc      map[string]requestHandler
	noticeFunc       map[string]requestHandler
	retainNoticeFunc map[string]requestHandler
}

// Name 返回模块名称
func (bll *Service) Name() string {
	return bll.cfg.getBase().module
}

// Load 初始化
func (bll *Service) Load(moduleName, moduleDesc, moduleVersion string, customSectionName string, config IConfig) {
	bll.requestFunc = map[string]requestHandler{}
	bll.noticeFunc = map[string]requestHandler{}
	bll.retainNoticeFunc = map[string]requestHandler{}
	bll.cfg = config
	if bll.cfg == nil {
		bll.cfg = &emptyConfig{}
	}
	// 设置模块信息
	bll.cfg.setBase(moduleName, moduleDesc, moduleVersion, customSectionName)
	// 加载配置
	loadConfig(bll.cfg)
}

func (bll *Service) bindRequest(route string, method requestHandler) {
	if bll.requestFunc == nil {
		bll.requestFunc = map[string]requestHandler{}
	}

	bll.requestFunc[route] = method
}

func (bll *Service) callRequest(pack easyCon.PackReq) ([]byte, easyCon.EResp, error) {
	if handler, ok := bll.requestFunc[pack.Route]; ok {
		return handler.Handle(pack.Content)
	}
	return nil, easyCon.ERespRouteNotFind, errors.New(fmt.Sprintf("route %s not find", pack.Route))
}

func (bll *Service) bindNotice(route string, method requestHandler, isRetain bool) {
	if isRetain == false {
		if bll.noticeFunc == nil {
			bll.noticeFunc = map[string]requestHandler{}
		}
		bll.noticeFunc[route] = method
	} else {
		if bll.retainNoticeFunc == nil {
			bll.retainNoticeFunc = map[string]requestHandler{}
		}
		bll.retainNoticeFunc[route] = method
	}
}

func (bll *Service) callNotice(pack easyCon.PackNotice, isRetain bool) {
	if isRetain == false {
		if handler, ok := bll.noticeFunc[pack.Route]; ok {
			_, _, _ = handler.Handle(pack.Content)
		}
	} else {
		if handler, ok := bll.retainNoticeFunc[pack.Route]; ok {
			_, _, _ = handler.Handle(pack.Content)
		}
	}
}

func (bll *Service) getBindCallback() (bool, bool, bool) {
	return len(bll.requestFunc) > 0, len(bll.noticeFunc) > 0, len(bll.retainNoticeFunc) > 0
}

func (bll *Service) config() IConfig {
	return bll.cfg
}

func (bll *Service) adapter() easyCon.IAdapter {
	return bll.adapterCon
}

func (bll *Service) setEnv(adapter easyCon.IAdapter) {
	bll.adapterCon = adapter
	for route := range bll.noticeFunc {
		bll.adapterCon.SubscribeNotice(route, false)
	}
	for route := range bll.retainNoticeFunc {
		bll.adapterCon.SubscribeNotice(route, true)
	}
}
