package input

import (
	"fmt"
	"os"
	"time"
)

// 定义一个 文件所需要的信息,这个也就是要收集的 日志文件
type File struct {
	File      *os.File    // 一个文件结构体，也就是要监听的文件
	FileInfo  os.FileInfo // 文件信息
	Path      string      // 文件路径
	FileState *FileState  // 文件状态
}

// 读取日志的事件信息
type FileEvent struct {
	ReadTime        time.Time          // 开始读取的时间
	Source          *string            // 源文件名
	InputType       string             // 输入类型
	DocumentType    string             // 文档类型
	Offset          int64              // 偏移量
	Bytes           int                // 读取大小
	Text            *string            // 读取到的文本信息
	Fields          *map[string]string // 自定义kv
	Fileinfo        *os.FileInfo       //日志信息
	IsPartial       bool               // 是否只读取局部信息
	fieldsUnderRoot bool               // 是否将自定义kv放在根
}

// 文件的状态信息
type FileState struct {
	Source      *string // 源地址，也就是 日志文件的地址
	Offset      int64   // 读取文件的偏移量
	FileStateOS *FileStateOS
}

// 检查一个文件是否是一个规则的文件
// Check that the file isn't a symlink, mode is regular or file is nil
func (f *File) IsRegularFile() bool {
	if f.File == nil {
		fmt.Println("Harvester: BUG: f arg is nil")
		return false
	}

	info, err := f.File.Stat()
	if err != nil {
		fmt.Println("File check fault: stat error: ", err.Error())
		return false
	}
	if !info.Mode().IsRegular() {
		fmt.Printf("Harvester: not a regular file: %q %s", info.Mode(), info.Name())
		return false
	}
	return true
}

// Check if the two files are the same
func (f1 *File) IsSameFile(f2 *File) bool {
	return os.SameFile(f1.FileInfo, f2.FileInfo)
}

// 检查给定的两个文件是否使用了同一个文件描述符
// IsSameFile checks if the given File path corresponds with the FileInfo given
func IsSameFile(path string, info os.FileInfo) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Error during file comparison: %s with %s - Error: %s", path, info.Name(), err)
		return false
	}
	return os.SameFile(fileInfo, info)
}
