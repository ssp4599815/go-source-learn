package harvester

import (
	"errors"
	"fmt"
	"github.com/ssp4599815/beat/filebeat/config"
	"github.com/ssp4599815/beat/filebeat/input"
	"golang.org/x/text/encoding"
	"io"
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

// 开始收集日志文件
func (h Harvester) Start() {
	// Starts harvester and picks the right type. In case type is not set, set it to default (log)
	go h.Harvest()  // 开启一个 goroutine 来收集日志
}

// 打开 h.Path 下的文件，并获取该文件描述符给 h.file
// open does open the files given under h.Path and assigns the file handler to h.file
func (h *Harvester) open() error {
	// 如果是 标准输入 这忽略
	// Special handing that "-" means to read from standard input
	if h.Path == "-" {
		h.file = os.Stdin
		return nil
	}

	for {
		var err error
		// 以只读的方式打开一个文件
		h.file, err = input.ReadOpen(h.Path)

		if err != nil {
			// retry on failure
			fmt.Printf("Failed opening %s: %s", h.Path, err)
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}

	// 将 该文件描述符 赋值为
	file := &input.File{
		File: h.file,
	}

	// Check we are not following a rabbit hole (symlinks ,etc.)
	if !file.IsRegularFile() {
		return errors.New("Given file is not a regular file.")
	}

	h.setFileOffset()

	return nil
}

// set the offset of the file to the right place. Takes configuation options into account
func (h *Harvester) setFileOffset() {
	if h.Offset > 0 {
		h.file.Seek(h.Offset, io.SeekStart) // 文件开头
	} else if h.Config.TailFiles {
		h.file.Seek(0, io.SeekEnd) // 文件末尾
	} else {
		h.file.Seek(0, io.SeekStart)
	}
}
