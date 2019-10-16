package logp

import (
	"fmt"
	"runtime"
	"strings"
)


type Logging struct {
	Selectors []string
	Files     *FileRotator
	ToSyslog  *bool // 日志输出到 syslog 中
	ToFiles   *bool // 日志输出到文件中
	Level     string
}

// 初始化日志系统
func Init(name string, config *Logging) error {
	logLevel, err := getLogLevel(config)
	if err != nil {
		return err
	}

	// 决定 日志是输出到 文件 还是 syslog
	var defaultToFiles, defaultToSyslog bool
	var defaultFilePath string

	if runtime.GOOS == "windows" {
		// always disables on windows
		defaultToSyslog = false
		defaultToFiles = true
		defaultFilePath = fmt.Sprintf("C:\\ProgramData\\%s\\Logs", name)
	} else {
		defaultToSyslog = true
		defaultToFiles = false
		defaultFilePath = fmt.Sprintf("/var/log/%s", name)
	}
	var toSyslog, toFiles bool
	if config.ToSyslog != nil {
		toSyslog = *config.ToSyslog
	} else {
		toSyslog = defaultToSyslog
	}
	if config.ToFiles != nil {
		toFiles = *config.ToFiles
	} else {
		toFiles = defaultToFiles
	}


}

// Priority 优先级
func getLogLevel(config *Logging) (Priority, error) {
	if config == nil || config.Level == "" {
		return LOG_ERR, nil
	}

	// 日志级别的对应关系
	levels := map[string]Priority{
		"critical": LOG_CRIT,
		"error":    LOG_ERR,
		"warning":  LOG_WARNING,
		"info":     LOG_INFO,
		"debug":    LOG_DEBUG,
	}

	// 获取日志级别
	level, ok := levels[strings.ToLower(config.Level)]
	if !ok {
		return 0, fmt.Errorf("unknown log level: %v", config.Level)
	}
	return level, nil
}
