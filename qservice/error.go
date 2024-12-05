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

var errorLogPath = "./log"

func GetErrors(moduleName string, lineCount int) string {
	//logFile := fmt.Sprintf("%s/%s/%s_%s_%s.log", errorLogPath, ym, day, moduleName, "Error")
	return ""
}

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
		nowTime := qconvert.DateTime.ToString(time.Now(), "yyyy-MM-dd HH:mm:ss")
		color.New(color.FgWhite).PrintfFunc()(nowTime)
		color.New(color.FgRed, color.Bold).PrintfFunc()(" [ERROR] %s", r)
		log += fmt.Sprintf("%s\n", r)
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
			log += fmt.Sprintf(" %s\n", lines[i])
		}

		// 执行外部方法
		if after != nil {
			after(log)
		}
	}
}

func writeErrLog(module, tp string, err string) {
	logStr := fmt.Sprintf("%s %s\n", qconvert.DateTime.ToString(time.Now(), "yyyy-MM-dd HH:mm:ss"), tp)
	logStr += fmt.Sprintf("%s\n", err)
	logStr += "----------------------------------------------------------------------------------------------\n\n"
	ym := qconvert.DateTime.ToString(time.Now(), "yyyy-MM")
	day := qconvert.DateTime.ToString(time.Now(), "dd")
	logFile := fmt.Sprintf("%s/%s/%s_%s_%s.log", errorLogPath, ym, day, module, "Error")
	logFile = qio.GetFullPath(logFile)
	_ = qio.WriteString(logFile, logStr, true)
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
