package harvester

import (
	"github.com/ssp4599815/beat/filebeat/config"
	"github.com/ssp4599815/beat/filebeat/input"
	"golang.org/x/text/encoding"
	"os"
	"time"
)

type Harvester struct {
	Path             string                  // the file path to harvest
	ProspectorConfig config.ProspectorConfig // prospector配置
	Config           *config.HarvesterConfig // harvester配置
	Offset           int64                   // 当前日志的偏移量
	FinishChan       chan int64              // 接受一个结束的信号
	SpoolerChan      chan *input.FileEvent   // 将 events 发送到 spooler 通道
	encoding         encoding.Encoding       // 日志文件的编码格式
	file             *os.File                // the file being watched  一个文件描述符，用于监听文件变化
	backoff          time.Duration           // 定义Filebeat在达到EOF之后再次检查文件之间等待的时间
}

// Interface for the different harvester types
type Typer interface {
	open()
	read()
}

// 启动一个 goroutine ,然后开始收集日志文件
func (h *Harvester) Start() {
	// Starts harvester and picks the right type. In case type is not set, set it to default (log)
	go h.Harvest() // 开启一个 goroutine 来收集日志
}
