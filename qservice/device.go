package qservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kamioair/qf/qdefine"
	"github.com/kamioair/qf/utils/qio"
	"runtime"
)

var DeviceCode deviceCode

type deviceCode struct {
}

// LoadFromFile 从文件中获取设备码
func (d *deviceCode) LoadFromFile() (qdefine.DeviceInfo, error) {
	file := getCodeFile()
	if qio.PathExists(file) {
		str, err := qio.ReadAllString(file)
		if err != nil {
			return qdefine.DeviceInfo{}, err
		}
		info := qdefine.DeviceInfo{}
		err = json.Unmarshal([]byte(str), &info)
		if err != nil {
			return qdefine.DeviceInfo{}, err
		}
		return info, nil
	}
	return qdefine.DeviceInfo{}, errors.New("deviceCode file not find")
}

// SaveToFile 将设备码写入文件
func (d *deviceCode) SaveToFile(info qdefine.DeviceInfo) error {
	// 写入文件
	file := getCodeFile()
	if file == "" {
		return errors.New("deviceCode file not find")
	}
	str, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	err = qio.WriteAllBytes(file, str, false)
	if err != nil {
		return err
	}
	return nil
}

func getCodeFile() string {
	root := qio.GetCurrentRoot()
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("%s\\Program Files\\Qf\\device", root)
	case "linux":
		return "/dev/qf/device"
	}
	return ""
}
