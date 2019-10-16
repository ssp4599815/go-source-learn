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
