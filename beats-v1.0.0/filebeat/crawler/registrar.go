package crawler

import (
	"encoding/json"
	"fmt"
	cfg "github.com/ssp4599815/beat/filebeat/config"
	"github.com/ssp4599815/beat/filebeat/input"
	. "github.com/ssp4599815/beat/filebeat/input"
	"os"
	"path/filepath"
)

// 用于记录日志读取时候的状态信息
type Registrar struct {
	// Registry 文件的路径位置
	registryFile string // path to the Registry file
	// 文件路径：文件状态 的对应关系
	State map[string]*FileState // map with all file paths inside and the corresponding（一致的）state
	//持久化文件状态用的一个管道，获取从  prospector 和 crawler 通道中的信息，然后发给 FileStates 来进行持久化
	Persist chan *input.FileState // channel used by the prospector and crawler to send FileStates to be persisted（持久化）
	running bool                  // 用来判断当前 registrar 是否在运行

	Channel chan []*FileEvent // 该通道用来获取 日志的事件信息，为后续进行持久化做准备
	done    chan struct{}     // 定义一个空的通道,用来 确认文件文件的持久化是否完成的
}

// 创建一个 registrar 对象
func NewRegistrar(registryFile string) (*Registrar, error) {
	r := &Registrar{
		registryFile: registryFile,
		done:         make(chan struct{}),
	}
	err := r.Init()
	return r, err
}

// registrar 的初始化
func (r *Registrar) Init() error {
	// Init state 初始化一些状态信息
	r.Persist = make(chan *FileState)      // 持久化时用的通道
	r.State = make(map[string]*FileState)  // 持久化文件的信息
	r.Channel = make(chan []*FileEvent, 1) // 获取日志文件的一个通道

	// 如不存在设置默认文件后缀 .filebeat
	// Set to default in case it is not set
	if r.registryFile == "" {
		r.registryFile = cfg.DefaultRegistryFile
	}

	// 确保记录持久化文件的目录是存在的
	// make sure the directory where we store the registryFile exists
	absPath, err := filepath.Abs(r.registryFile)
	if err != nil {
		return fmt.Errorf("Failed to get the absolute path of %s: %v ", r.registryFile, err)
	}
	r.registryFile = absPath

	// 如果不存在的话就创建
	// Create directory if it does not already exist.
	registrtyPath := filepath.Dir(r.registryFile)
	err = os.MkdirAll(registrtyPath, 0755)
	if err != nil {
		return fmt.Errorf("Failed to created registry file dir %s: %v\n", registrtyPath, err)
	}

	return nil
}

//  从配置的 RegistryFile 文件里， 获取当前 读取文件的状态信息
// loadState fetches the previous reading state from the configure RegistryFile file
// The default file is .filebeat file which is stored in the same path as the binary is running
func (r *Registrar) LoadState() {
	if existing, e := os.Open(r.registryFile); e == nil {
		defer existing.Close()
		fmt.Printf("Loading registrar data from %s", r.registryFile)

		// 将持久化的文件状态信息（json格式） 解析为 map 对象
		decoder := json.NewDecoder(existing)
		decoder.Decode(&r.State)
	}
}

// 停止 registrar
func (r *Registrar) Stop() {

}

// 获取文件的状态 offset
// - 如果是老文件 就返回当前文件的 lastState
// - 如果是新文件，就返回 0
func (r *Registrar) fetchState(filePath string, fileInfo os.FileInfo) (int64, bool) {
	// check if there is a state for this file
	lastState, isFound := r.GetFileState(filePath)

	if isFound && input.IsSameFile(filePath, fileInfo) {
		fmt.Println("registar, Same file as before found, Fetch the state and persist it.")
		// We're resuming - throw the last state back downstaream so wo resave it
		// And retuen the offset - also force harvest in case the file is old and we're about to skip it
		r.Persist <- lastState
		return lastState.Offset, true
	}

	if previous, err := r.getPreviousFile(filePath, fileInfo); err != nil {
		// File has rotated betewwn shutdown and startup
		// We return last state downstream, with a modified event source with the new file name
		// And return the offset - also force harvest in case the file is old and we're about to skip it
		fmt.Printf("Detected rename of a previously harvested file: %s -> %s", previous, filePath)

		lastState, _ := r.GetFileState(previous)
		lastState.Source = &filePath
		r.Persist <- lastState
		return lastState.Offset, true
	}

	if isFound {
		fmt.Println("Not resuming rotated file: ", filePath)
	}
	// New file so just start from an automatic position
	return 0, false
}

func (r *Registrar) GetFileState(path string) (*FileState, bool) {
	state, exist := r.State[path]
	return state, exist
}

// 核查 registrar  是否一个新文件已经存在了，只是使用了不同的名称（也就是使用了同一个文件描述符）
// 一旦一个老的文件被发现了，就直接返回该文件，如果不是就返回错误
// getPreviousFile checks in the registrar if there is the newFile already exist with a different name
// In case an old file is found, the path to the file is retuened, if not, an error is returned
func (r *Registrar) getPreviousFile(newFilePath string, newFileInfo os.FileInfo) (string, error) {
	newState := input.GetOSFileState(&newFileInfo)
	for oldFilePath, oldState := range r.State {

		// skipping when path the same
		if oldFilePath == newFilePath {
			continue
		}

		// Compare states
		if newState.IsSame(oldState.FileStateOS) {
			fmt.Printf("Old file with new name found: %s is no %s", oldFilePath, newFilePath)
			return oldFilePath, nil
		}
	}
	return "", fmt.Errorf("No previous file found")
}
