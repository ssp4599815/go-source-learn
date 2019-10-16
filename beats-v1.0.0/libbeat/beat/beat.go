package beat

import (
	"fmt"
	"github.com/ssp4599815/beat/libbeat/cfgfile"
	"github.com/ssp4599815/beat/libbeat/outputs"
	"github.com/ssp4599815/beat/libbeat/publisher"
	"github.com/ssp4599815/beat/libbeat/service"
	"log"
	"os"
)

// 定义了一个公共的接口，只要所有的 beat 实现了这几个接口就可以收集日志了，也很方便的进行后期扩展
// Beater interface that every beat must use
type Beater interface {
	Config(*Beat) error  // beat初始化配置文件时调用
	Setup(*Beat) error   // beat其中的时候调用
	Run(*Beat) error     // beat运行时调用
	Cleanup(*Beat) error // beat退出时执行清理工作
	Stop()               // 停止beat
}

// 定义一个 beat所需要的信息
// Basic beat information
type Beat struct {
	Name    string      // beat的名称
	Version string      // beat的版本
	Config  *BeatConfig // beat的配置,解析出来的配置文件会放在这里
	BT      Beater      // 这里就是每一个要实现的beat接口
	Events  publisher.Client
}

// 针对每一个 beat的基础配置
// Basic configuration of every beat
type BeatConfig struct {
	Output map[string]outputs.MothershipConfig // 日志的输出
	// Logging logp.Logging                        // 记录log
	Shipper publisher.ShipperConfig // 消费者
}

// 初始化一个 beat 对象
// Initiates a new beat object
func NewBeat(name string, version string, bt Beater) *Beat {
	b := Beat{
		Version: version,
		Name:    name,
		BT:      bt, // 传进来的 beat
	}
	return &b
}

// 初始化配置文件，并从 Beat.Config 读取 默认的配置信息
// LoadConfig inits the config file and reads the default config information
// into Beat.Config. It exists the processes in case of errors
func (b *Beat) LoadConfig() {
	// 读取配置文件
	err := cfgfile.Read(&b.Config, "")
	if err != nil {
		fmt.Printf("Loading config file error: %v\n", err)
		os.Exit(1)
	}
	// 初始化log

	// 初始化 publisher

}

func (b *Beat) Run() {
	// Setup beater object
	err := b.BT.Setup(b)
	if err != nil {
		log.Fatal("Setup returned an error: ", err)
	}

	// 截获退出信号并执行相应的退出函数
	// callback is called if the processes is asked to stop
	// this needs to be called before the main loop is started so that
	// it can register tie signals that stop or query the loop
	service.HandleSignals(b.BT.Stop)

	fmt.Printf("%s successfully setup. Start running.", b.Name)

	// Run beater specific stuff  运行指定的 beater
	err = b.BT.Run(b)
	if err != nil {
		log.Fatal("Run returned an error: ", err)
	}

	fmt.Printf("Cleaning up %s before shutting down.", b.Name)

	// Call beater cleanup function
	err = b.BT.Cleanup(b)
	if err != nil {
		log.Fatal(err)
	}
}

// Stop calls the beater Stop action
func (b *Beat) Stop() {
	b.BT.Stop()
}
