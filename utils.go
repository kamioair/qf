package qf

import (
	"encoding/json"
	"fmt"
	"github.com/kamioair/utils/qconvert"
	"github.com/kamioair/utils/qio"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

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
