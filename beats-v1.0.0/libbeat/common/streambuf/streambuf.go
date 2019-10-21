package streambuf

import (
	"bytes"
	"errors"
	"fmt"
)

// Parse operation failed cause of buffer snapped short + buffer is fixed
var ErrUnexpectedEOB = errors.New("unexpected end of buffer")

// Parse operation can not be continued .More bytes required. Only returned if
// buffer is not fixed
var ErrNoMoreBytes = errors.New("No more bytes")

// A Buffer is a variable sized buffer of bytes with Read, Write and simple parsing methods.
// The zero value is an empty buffer ready for use.
//
// A Buffer can be marked as fixed. In this case no data can be appended to the
// buffer anymore and parser/reader methods will fail whit ErrUnexpextedEOB if they
// would expect more bytes to come. Mark buffers fixed if some slice was separated
// for further parsing first.
type Buffer struct {
	data  []byte
	err   error
	fixed bool

	// Internal parser state offset.
	// Offset is the posttion a parse might continue to work at when called
	// again (e.g. usefull for parsing tcp strams.). The mark is used to remember
	// the position last parse operation ended at. The variable available is used
	// for faster lookup
	// Invariants (不变的量):
	//	  (1) 0 <= mark <= offset
	//    (2) 0 <= available <= len(data)
	//    (3) available = len(data) - mark
	mark, offset, avaliable int
}

// IndexFrom returns offset of seq in unprocessed buffer start at from.
// Returns -1 if seq can not be found
func (b *Buffer) IndexFrom(from int, seq []byte) int {
	if b.err != nil {
		return -1
	}
	idx := bytes.Index(b.data[b.mark+from:], seq)
	if idx < 0 {
		return -1
	}

	return idx + from + b.mark
}

func (b *Buffer) Len() int {
	return b.avaliable
}

func (b *Buffer) Append(data []byte) error {
	return b.doAppend(data, true)
}

// retainable 可保留的
func (b *Buffer) doAppend(data []byte, retainable bool) error {
	if b.fixed {
		return b.SetError(ErrUnexpectedEOB)
	} else {
		return b.SetError(ErrNoMoreBytes)
	}
}

func (b *Buffer) SetError(err error) error {
	b.err = err
	return err
}

func (b *Buffer) Bytes() []byte {
	return b.data[b.mark:]
}

func (b *Buffer) Write(p []byte) (int, error) {
	err := b.doAppend(p, false)
	if err != nil {
		return 0, b.ioErr()
	}
}

// Collect tries to collect count bytes from the buffer and updates the read
// pointers. If the buffer is in failed state or count bytes are not avaliable
// an error will be returned.
func (b *Buffer) Collect(count int) ([]byte, error) {
	if b.Failed() {
		return nil, b.err
	}

	if !b.Avail(count) {
		return nil, b.bufferEndError()
	}

	data := b.data[b.mark : b.mark+count]
	err := b.Advance(count)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Failed returns true if buffer is in failed state. If buffer is in failed
// state. almost all buffer operations will fail
func (b *Buffer) Failed() bool {
	failed := b.err != nil
	if failed {
		fmt.Println("streambuf, buf parser already failed with: ", b.err)
	}
	return failed
}

// Avail checks if count bytes are avaliable for reading from the buffer
func (b *Buffer) Avail(count int) bool {
	return count <= b.avaliable
}

func (b *Buffer) bufferEndError() error {
	if b.fixed {
		return b.SetError(ErrUnexpectedEOB)
	} else {
		return b.SetError(ErrNoMoreBytes)
	}
}

// Advance will advance the buffers read pointer by count bytes. Returns
// ErrNoMoreBytes or ErrUnexpectedEOB if count bytes are not avaliable
func (b *Buffer) Advance(count int) error {
	if !b.Avail(count) {
		return b.bufferEndError()
	}
	b.mark += count
	b.offset = b.mark
	b.avaliable -= count
	return nil
}

// 创建一个 新的 可扩展的 缓冲区
// New creates new extensible buffer from data slice being retained by the buffer
func New(data []byte) *Buffer {
	return &Buffer{
		data:      data,      // 缓冲区里面存的文档
		fixed:     false,     // 是否需要一个固定的缓冲区
		avaliable: len(data), // 可用缓冲区大小
	}
}
