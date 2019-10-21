package harvester

import (
	"github.com/ssp4599815/beat/libbeat/common/streambuf"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
	"io"
	"time"
)

// 跟踪最近一次从底层 reader 读取字节的时间
// timeReader keeps track of last time bytes have been read from underlying reader
type timeReader struct {
	reader io.Reader // 一个 reader ，用来读取当前的文件
	// 从 input steam 中读取数据的 最近一次的时间
	lastReadTime time.Time // last time we read some data from input steam
}

// 从底层 reader 中读取一行，使用配置好的 codec 解码从 input steam 中读取到的每一行文件
// reader 跟踪每个解码后的行 从 原始输入流 消耗的字节数
// lineReader reads lines from underlying reader, decoding the input stream
// using the configured codec. the reader keeps track of bytes comsumed （消耗）
// from raw input stream for every decoded line.
type lineReader struct {
	rawInput   io.Reader         // input reader
	codec      encoding.Encoding // 当前文件的编码格式
	bufferSize int               // 缓冲区大小

	nl        []byte
	inBuffer  *streambuf.Buffer
	outBuffer *streambuf.Buffer
	inOffset  int // input buffer read offset
	byteCount int // number of bytes decoded from input buffer into output buffer
	decoder   transform.Transformer
}

const maxConsecutiveEmptyReads = 100 // 最大的连续空行读

// 必须要实现一个 Read  timeIn 参数才可以使用，也就是想当时 实现了 io.Reader 中的 Read 方法
func (r *timeReader) Read(p []byte) (int, error) {
	var err error
	n := 0
	// 用来读取空行的
	for i := maxConsecutiveEmptyReads; i > 0; i-- {
		n, err := r.reader.Read(p) // 将读取到的文件 放到 p 里面
		if n > 0 { // 如果读取到文件 就跳出循环
			r.lastReadTime = time.Now()
			break
		}
		if err != nil {
			break
		}
	}
	return n, err
}

// 创建一个新的 TimeReader 对象
func newTimedReader(reader io.Reader) *timeReader {
	r := &timeReader{
		reader: reader,
	}
	return r
}

// 创建一个 新的 LineReader 对象
func newLineReader(input io.Reader, codec encoding.Encoding, bufferSize int) (*lineReader, error) {
	l := &lineReader{}
	// 初始化一个 LineReader
	if err := l.init(input, codec, bufferSize); err != nil {
		return nil, err
	}
	return l, nil
}

// 初始化一个 lineReader
func (l *lineReader) init(input io.Reader, codec encoding.Encoding, bufferSize int) error {
	l.rawInput = input        // input reader 对象,就是要去读的文件的对象
	l.codec = codec           // 文件的编码格式
	l.bufferSize = bufferSize // 缓冲区大小

	l.codec.NewEncoder() // 创建一个新的 encoder 编码器
	// 将文件进行编码，并返回编码后的结果
	nl, _, err := transform.Bytes(l.codec.NewEncoder(), []byte{'\n'})
	if err != nil {
		return err
	}

	l.nl = nl                        // 编码后的文档
	l.decoder = l.codec.NewDecoder() // 创建一个解码器
	l.inBuffer = streambuf.New(nil)  // 创建一个输入文件的缓冲区
	l.outBuffer = streambuf.New(nil) // 创建一个输出文件的缓冲区
	return nil
}

func (l *lineReader) next() ([]byte, int, error) {
	for {
		// read next 'potential'（潜在的） line from input buffer/reader
		err := l.advance()
		if err != nil {
			return nil, 0, err
		}

		// check last decoded byte really being '\n'
		buf := l.outBuffer.Bytes()
		if buf[len(buf)-1] == '\n' { // 检查 是否已经读取到了文件的末尾
			break
		}
	}

	// output buffer contains complate line ending with '\n' . Extaract
	// byte slice from buffer and reset output buffer
	bytes, err := l.outBuffer.Collect(l.outBuffer.Len())
	l.outBuffer.Reset()
	if err != nil {
		panic(err)
	}

	// return and reset consumeed bytes count
	sz := l.byteCount
	l.byteCount = 0
	return bytes, sz, nil
}

func (l *lineReader) advance() error {
	var idx int
	var err error

	// fill inBuffer until '\n' sequence has been found in input buffer
	for {
		idx = l.inBuffer.IndexFrom(l.inOffset, l.nl)
		if idx >= 0 {
			break
		}
		if err != nil {
			// if no newline and last read returned error, return error now
			return err
		}

		// increase search offset to reduce iterations on buffer when looping
		newOffset := l.inBuffer.Len() - len(l.nl)
		if newOffset > l.inOffset {
			l.inOffset = newOffset
		}

		// try to read more bytes into buffer
		n := 0
		buf := make([]byte, l.bufferSize)
		n, err = l.rawInput.Read(buf)
		l.inBuffer.Append(buf[:n])
		if n == 0 && err != nil {
			// return error only if no bytes have been received. Otherwise try to
			// parse '\n' before returning the error.
			return err
		}

		// empty read => return buffer error (more bytes required error)
		if n == 0 {
			return streambuf.ErrNoMoreBytes
		}
	}

	// found encoded byte sequence for '\n' in buffer
	// -> decode input sequence into outBuffer
	sz, err := l.decode(idx + len(l.nl))
}

func (l *lineReader) decode(end int) (int, error) {
	var err error
	buffer := make([]byte, 1024)
	inBytes := l.inBuffer.Bytes()
	start := 0

	for start < end {
		var nDst, nSrc int

		nDst, nSrc, err = l.decoder.Transform(buffer, inBytes[start:end], false)
		start += nSrc

		l.outBuffer.Write(buffer[:nDst])
	}
}

// partial returns current state of decoded input bytes and amount of bytes
// processed so far. If decoder has detected an error in input stream, the error
// will be returned
func (l *lineReader) partial() ([]byte, int, error) {
	// decode all input buffer
	sz, err := l.decode(l.inBuffer.Len())
	l.inBuffer.Advance(sz)
	l.inBuffer.Reset()

	l.inOffset -= sz
	if l.inOffset < 0 {
		l.inOffset = 0
	}

	// return current state of outBuffer, but do not consume any content yet
	bytes := l.outBuffer.Bytes()
	sz = l.byteCount
	return bytes, sz, err
}
