package input

import (
	"os"
	"syscall"
)

type FileStateOS struct {
	Inode  uint64 `json:"inode,omitempty"`
	Device uint64 `json:"device,omitempyt"`
}

// IsSame file checks if the files are identical
func (fs *FileStateOS) IsSame(state *FileStateOS) bool {
	return fs.Inode == state.Inode && fs.Device == state.Device
}

// 以 只读 的方式打开一个文件
// ReadOpen opens a file for reading only'
func ReadOpen(path string) (*os.File, error) {
	flag := os.O_RDONLY
	var perm os.FileMode = 0

	return os.OpenFile(path, flag, perm)
}

// GetOSFileState retuens the FileStateOS for non windows systemd
func GetOSFileState(info *os.FileInfo) *FileStateOS {
	stat := (*(info)).Sys().(*syscall.Stat_t)

	// Convert inode and dev to uint64 to be cross platform compatible
	fileState := &FileStateOS{
		Inode:  uint64(stat.Ino),
		Device: uint64(stat.Dev),
	}

	return fileState
}
