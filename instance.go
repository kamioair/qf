package qf

var instance IService

// LogDebug 发送Debug日志
func LogDebug(content string) {
	if instance == nil {
		return
	}

	instance.SendLogDebug(content)
}

// LogWarn 发送Warn日志
func LogWarn(content string) {
	if instance == nil {
		return
	}

	instance.SendLogWarn(content)
}

// LogError 发送Error日志
func LogError(content string, err error) {
	if instance == nil {
		return
	}

	instance.SendLogError(content, err)
}
