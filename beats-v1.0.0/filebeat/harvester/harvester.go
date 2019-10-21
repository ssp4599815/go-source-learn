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

// 启动一个 goroutine ,然后开始收集日志文件
func (h *Harvester) Start() {
	// Starts harvester and picks the right type. In case type is not set, set it to default (log)
	go h.Harvest() // 开启一个 goroutine 来收集日志
}

// 打开 h.Path 下的文件，并获取该文件描述符给 h.file，然后设置 该文件 要读取的位置
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
			// 如果打开失败，就 sleep 5秒后 继续打开文件，知道打开为止
			// retry on failure
			fmt.Printf("Failed opening %s: %s", h.Path, err)
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}

	// 将 该文件描述符 赋值给 input
	file := &input.File{
		File: h.file,
	}

	// 判断我们要收集的日志 是否是一个符合规则的 文件
	// Check we are not following a rabbit hole (symlinks ,etc.)
	if !file.IsRegularFile() {
		return errors.New("Given file is not a regular file.")
	}

	// 设置 要读取文件的 offset ,是从文件的开头 还是结尾 还是其他情况
	h.setFileOffset()

	return nil
}

// set the offset of the file to the right place. Takes configuation options into account
func (h *Harvester) setFileOffset() {
	if h.Offset > 0 {
		_, _ = h.file.Seek(h.Offset, io.SeekStart) // 如果 h.Offset 有记录的话，证明是重新启动后重新读取的该文件，需要从当前位置开始读
	} else if h.Config.TailFiles {
		_, _ = h.file.Seek(0, io.SeekEnd) // 如果是 tail file  的方式， 就从文件末尾开始读取文件
	} else {
		_, _ = h.file.Seek(0, io.SeekStart) // 都不是的话，就从文件的开头开始 ，把所有的文件内容都读取到
	}
}
