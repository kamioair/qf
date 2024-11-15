package qservice

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/kamioair/qf/utils/qconvert"
	"github.com/kamioair/qf/utils/qio"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Recover
//
//	@Description: Panic的异常收集
func errRecover(after func(err string)) {
	if r := recover(); r != nil {
		// 获取异常
		var buf [4096]byte
		n := runtime.Stack(buf[:], false)
		stackInfo := string(buf[:n])

		// 输出异常
		log := ""
		fmt.Println("")
		nowTime := qconvert.DateTime.ToString(time.Now(), "yyyy-MM-dd HH:mm:ss")
		color.New(color.FgWhite).PrintfFunc()(nowTime)
		log += nowTime
		color.New(color.FgRed, color.Bold).PrintfFunc()(" [ERROR] %s", r)
		log += fmt.Sprintf(" [ERROR] %s\n", r)
		fmt.Println("")
		lines := strings.Split(stackInfo, "\n")
		for i := 0; i < len(lines); i++ {
			line := strings.Replace(lines[i], "\t", "", -1)
			if strings.HasPrefix(line, "panic") {
				errStr := ""
				if i+3 < len(lines) {
					errStr += formatStack("curr", lines[i+2], lines[i+3])
				}
				if i+5 < len(lines) {
					errStr += formatStack("upper", lines[i+4], lines[i+5])
				}
				color.New(color.FgMagenta).PrintfFunc()("%s\n", errStr)
			}
			log += fmt.Sprintf("%s\n", lines[i])
		}
		// 写入日志
		logFile := fmt.Sprintf("%s/%s_Error.log", "./log", qconvert.DateTime.ToString(time.Now(), "yyyy-MM-dd"))
		logFile = qio.GetFullPath(logFile)
		log += "----------------------------------------------------------------------------------------------\n\n"
		_ = qio.WriteString(logFile, log, true)

		// 执行外部方法
		if after != nil {
			after(fmt.Sprintf("%s", r))
		}
	}
}

func formatStack(flag, name string, row string) string {
	sp := strings.Split(strings.Replace(row, "\t", "", -1), "+")
	funcName := filepath.Base(name)
	matches := regexp.MustCompile(`\((.*?)\)`).FindAllStringSubmatch(funcName, -1)
	if matches != nil && len(matches) > 0 && len(matches[len(matches)-1]) > 0 {
		funcName = strings.Replace(funcName, matches[len(matches)-1][0], "(...)", 1)
	}
	return fmt.Sprintf("   %s\n      %s\n", funcName, sp[0])
}
