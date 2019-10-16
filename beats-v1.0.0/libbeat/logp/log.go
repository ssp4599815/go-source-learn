package logp

import "log"

type Priority int

// 定义日志级别
const (
	LOG_EMERG Priority = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

type Logger struct {
	toFile bool
	level  Priority

	logger  *log.Logger
	rotator *FileRotator
}

var _log Logger

func LogInit(level Priority, prefix string) {

	_log.level = level
	
}
