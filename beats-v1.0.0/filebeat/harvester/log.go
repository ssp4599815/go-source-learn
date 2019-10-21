package harvester

import (
	"fmt"
	"github.com/ssp4599815/beat/filebeat/config"
	"github.com/ssp4599815/beat/filebeat/input"
	"io"
	"time"
)

// 创建一个新的 harvester,用来收集日志，并将收集到的日志 发动到 spooler 中
func NewHarvester(prospectorCfg config.ProspectorConfig, cfg *config.HarvesterConfig, path string, signal chan int64, spooler chan *input.FileEvent) (*Harvester, error) {
	// 获取日志的编码格式， utf-8 gbk....
	encoding, ok := findEncoding(cfg.Encoding)
	if !ok || encoding == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", cfg.Encoding)
	}

	// 初始化一个 harvester
	h := &Harvester{
		Path:             path,          // 要收集的日志文件路径
		ProspectorConfig: prospectorCfg, // prospector 配置
		Config:           cfg,           // Harvester 配置
		FinishChan:       signal,        // 接受关闭的信号的通道
		SpoolerChan:      spooler,       // 将收集到的日志放到 spooler 中
		encoding:         encoding,      // 文件的编码格式
		backoff:          prospectorCfg.Harvester.BackoffDuration,
	}
	return h, nil
}

// 一行一行的读取日志，并且将读取到的信息发送到 SpoolerChan 中
// Log harvester reads files line by line and send events to the defined output
func (h *Harvester) Harvest() {
	// 打开 h.Path 下的文件，并获取该文件描述符给 h.file
	err := h.open()

	// 延迟关闭
	defer func() {
		// on completion,push offset so we can continue where we left off if we relaunch on the same file
		// 一旦完成，将当时文件的偏移量保存下来，使得重启后能读取到同样的文件位置
		h.FinishChan <- h.Offset
		// Make sure file is closed as soon as harvester exits
		_ = h.file.Close()
	}()

	if err != nil {
		fmt.Println("Stop harvesting. Unexpected Error: ", err)
		return
	}
	// 获取文件的状态信息
	info, err := h.file.Stat()
	if err != nil {
		fmt.Println(" Stop Harvesting. Unexpected Error: ", err)
		return
	}

	fmt.Println("Harvester started for file: ", h.Path)

	// 每次启动的时候，都要初始化 offset 信息
	// Load last offset from registrar
	h.initOffset()

	// 最近一次从 底层 reader (h.file) 读取字节的时间
	// timeIn 实现了 io.Reader() 接口，可以使用 timeIn.Read() 来去读文件 h.file
	timeIn := newTimedReader(h.file)

	// 创建一个 新的 LineReader 对象
	reader, err := newLineReader(timeIn, h.encoding, h.Config.BufferSize)
	if err != nil {
		fmt.Printf("Stop Harvesting. Unexpected Error: %s", err)
		return
	}

	// XXX: lastReadTime handling last time a full line was read only?
	//      timeReader provieds timestamp some bytes have actually been read from file
	// 最后一次读取文件的时间
	lastReadTime := time.Now()

	// remember size of last partial line being sent. Do not publish partial line, if
	// no new bytes have been processed
	// 记住 最后一次发送文件的大小
	lastPartialLen := 0

	for {
		// 获取 读取到的文本，读取到文本的大小
		text, bytesRead, isPartial, err := readLine(reader, &timeIn.lastReadTime, h.Config.PartialLineWatingDuration)

		if err != nil {
			// In case of err = io.EOF returns nil
			err = h.handleReadlineError(lastReadTime, err)
			if err != nil {
				fmt.Printf("File reading error. Stopping harvester. Error: %s", err)
				return
			}
			continue
		}

		lastReadTime = time.Now()

		// Reset Backoff
		// Filebeat检测到某个文件到了EOF（文件结尾）之后，每次等待多久再去检测文件是否有更新，默认为1s
		h.backoff = h.Config.BackoffDuration

		if isPartial {
			if bytesRead <= lastPartialLen {
				// drop partial line event, as no new bytes have been consumed from imput stream
				continue
			}
			lastPartialLen = bytesRead
		} else {
			lastPartialLen = 0
		}

		// Sends text to spooler
		event := &input.FileEvent{
			ReadTime:     lastReadTime,
			Source:       &h.Path,
			InputType:    h.Config.InputType,
			DocumentType: h.Config.DocumentType,
			Offset:       h.Offset,
			Bytes:        bytesRead,
			Text:         &text,
			Fields:       &h.Config.Fields,
			Fileinfo:     &info,
			IsPartial:    isPartial,
		}

		if !isPartial {
			h.Offset += int64(bytesRead) // Update offset if complete line has been processed
		}

		event.SetFieldsUnderRoot(h.Config.FieldsUnderRoot)
		h.SpoolerChan <- event // ship the new event downstream
	}
}

// 初始化 要读取文件的偏移量
// initOffset finds the current offset of the file and sets it in the harvester as postition
func (h *Harvester) initOffset() {
	// 获取文件的偏移量
	// get current offset in file
	offset, _ := h.file.Seek(0, io.SeekCurrent) // 获取当前位置的偏移量

	if h.Offset > 0 {
		fmt.Printf("harvester, harvest: %q position: %d (offset snapshot: %d)", h.Path, h.Offset, offset)
	} else if h.Config.TailFiles {
		fmt.Printf("harvester, harvest: (tailing) %q (offset snapshot: %d)", h.Path, offset)
	} else {
		fmt.Printf("harvester, harvest: %q (offset snapshot:%d)", h.Path, offset)
	}
	h.Offset = offset // 将当前文件的偏移量 复制到  harvester.Offset 中,后面再读取的时候会使用
}

// 公共函数
/*** Utility Functions ***/

// 读取一整行并放入到 buffer 中
// 为了防止读取到不完整的行，readLine 会等待 partialLineWaiting 的时间，是为了这段时间内可以接收到新的 日志片段
// readLine reads a full line into buffer and returns it
// In case of partial lines, readLine waits for a maximum of partialLineWaiting seconds for new segments to arrive
// This could potentialy（潜在的） be improved / replaced by https://github.com/elastic/beats/libbeat/tree/master/common/streambuf
func readLine(reader *lineReader, lastReadTime *time.Time, partialLineWaiting time.Duration) (string, int, bool, error) {
	for {
		line, sz, err := reader.next()
		if err != nil {
			if err == io.EOF { // 如果是读取到了行尾部
				return "", 0, false, err // text, bytesRead, isPartial, err
			}
		}
		if sz != 0 { // 如果读取了完整了一行
			return readlineString(line, sz, false)
		}

		// test for no file updates longer than partialLineWaiting
		if time.Since(*lastReadTime) >= partialLineWaiting {
			// return all bytes read for current line to be processed
			// line might grow with furture read attempts
			line, sz, err = reader.partial()
			return readlineString(line, sz, true)
		}

		// wait for file updates before reading new lines
		time.Sleep(1 * time.Second)
	}
}

// 检测 给定个一行 是否是完整的一个一行 ，以 \n 结尾
// isLine checks if the given byte array is a line, means has a line ending \n
func isLine(line []byte) bool {
	if line == nil || len(line) == 0 { // 不是一行, 这是一个空行
		return false
	}
	if line[len(line)-1] != '\n' { // 行位是不是 '\n' 结尾，如果不是 那就不是一个完整的行
		return false
	}
	return true
}

// 检测 给定的行的结尾 的编码格式 ，是 \n 还是 \r\n
// \n 返回字节为 1
// \r\n 返回的字节为 2
// 其他返回 0
// lineEndingChars returns the number of lines ending chars the given by array has
// In case of Unix/Linux files, it is -1, incase of Windows mostly -2
func lineEndingChars(line []byte) int {
	if isLine(line) {
		return 0
	}
	if line[len(line)-1] == '\n' { // Unix/Linux 每一行的结尾是 '\n'
		if len(line) > 1 && line[len(line)-2] == '\r' { // windows 每一行的结尾是 '\r\n'
			return 2
		}

		return 1
	}
	return 0
}

// 将读取到的 []byte 转换为 string 类型
func readlineString(bytes []byte, sz int, partial bool) (string, int, bool, error) {
	// 将行末尾的 '\n' 或者 '\r\n' 去掉
	s := string(bytes)[:len(bytes)-lineEndingChars(bytes)]
	return s, sz, partial, nil // text, bytesRead, isPartial, err
}
