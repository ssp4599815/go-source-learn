package input

import "os"

// 以 只读 的方式打开一个文件
// ReadOpen opens a file for reading only'
func ReadOpen(path string) (*os.File, error) {
	flag := os.O_RDONLY
	var perm os.FileMode = 0

	return os.OpenFile(path, flag, perm)
}
