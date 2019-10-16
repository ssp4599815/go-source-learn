package beat

import (
	"fmt"
	"os"

	"github.com/ssp4599815/beat/libbeat/beat"
	"github.com/ssp4599815/beat/libbeat/cfgfile"

	cfg "github.com/ssp4599815/beat/filebeat/config"
	. "github.com/ssp4599815/beat/filebeat/crawler"
	. "github.com/ssp4599815/beat/filebeat/input"
)

// Filebeat 定义一个 filebeat 所需要的信息
type Filebeat struct {
	FbConfig      *cfg.Config       // filebeat 配置文件
	publisherChan chan []*FileEvent // 是一个channel， 把从 harvesters 读取到的日志发送到 spooler
	Spooler       *Spooler          // 把从 通道里读取日志缓存起来，等待 publisher来拉取
	registrar     *Registrar        // 记录每次读取文件的状态信息
}

// 加载所有的配置文件
// Config setup up the filebeat configuration by fetch all additional config files
func (fb *Filebeat) Config(b *beat.Beat) error {
	// Load Base config  加载基础的配置文件
	err := cfgfile.Read(&fb.FbConfig, "")
	if err != nil {
		return fmt.Errorf("Error reading config file: %v\n", err)
	}

	// 如果 config_dir 指定的话，就拉取所有的配置文件
	// Check if optional config_dir is set to fetch additional prospecrot config file
	fb.FbConfig.FetchConfigs()

	return nil
}

// 启动程序时要做的操作
func (fb *Filebeat) Setup(b *beat.Beat) error {
	return nil
}

// 正式运行filebeat
func (fb *Filebeat) Run(b *beat.Beat) error {
	// 处理异常情况
	defer func() {
		p := recover()
		if p == nil {
			return
		}

		fmt.Printf("recovered panic:%v", p)
		os.Exit(1)
	}()

	var err error

	// 初始化通道，该通道是将获取到的 event 发送到 publisher
	fb.publisherChan = make(chan []*FileEvent, 1)

	// 开启一个 registrar 来持久化 文件状态
	// setup registrar to persist state
	fb.registrar, err = NewRegistrar(fb.FbConfig.Filebeat.RegistryFile)
	if err != nil {
		fmt.Printf("Could not init registrar: %v", err)
		return err
	}

	// 开启一个爬虫，来抓取 日志文件
	crawl := &Crawler{
		Registrar: fb.registrar,
	}

	// 启动 crawer 的时候，从 持久化文件中 加载 当前日志文件的状态信息， 后续给 prospector 使用
	// Load the previous log file locations now ,for use in prospector
	fb.registrar.LoadState()

	// 初始化并启动 spooler: 从harvesters 获取 日志的事件信息 放到缓冲区里面，然后定期的 通过通道传递给 publisher
	// Init and start spooler: harvesters dump events into the spooler
	fb.Spooler = NewSpooler(fb)
	// 配置 spooler的相关信息
	err = fb.Spooler.Config()
	if err != nil {
		fmt.Printf("Could not init spooler ：%v", err)
		return err
	}

	// 启动 spooler 开始监听来自 harvester Channel的通道信息
	// start up spooler
	go fb.Spooler.Run()

	// 准备开始探测文件（从 配置文件的 input 中获取的所有要收集的日志）
	// Prospectors 为所有的 prospect
	crawl.Start(fb.FbConfig.Filebeat.Prospectors, fb.Spooler.Channel)

	// 处理通道中的 日志事件信息 然后交给 output
	// Publishes event to output
	go Publish(b, fb)

	// 开启 registrar，用来记录所有监听文件的 最后一次确认的位置。
	// registrar records last acknowledged positions in all files.
	fb.registrar.Run()

	return nil
}

// 执行一些 退出时的清理工作
func (fb *Filebeat) Cleanup(b *beat.Beat) error {
	return nil
}

// 清理工作完成后 执行退出
// Stop is called on exit for cleanup
func (fb *Filebeat) Stop() {
	// 主要做一些停止时候的清理工作

	// Stop harvesters
	// Stop prospectors

	// Stopping spooler will flush items
	fb.Spooler.Stop()

	// Stopping registrar will write last state
	fb.registrar.Stop()

	// Close channels
	//close(fb.publisherChan)
}

// 将收集的日志事件信息传递出去
func Publish(beat *beat.Beat, fb *Filebeat) {
	fmt.Println("Start sending events to output")

	// 从 spool 中获取日志的事件信息，并刷新到output中
	// Receives events from spool during flush
	for events := range fb.publisherChan {

		pubEvents := make([]common.MapStr, 0, len(events))
		for _, event := range events {
			pubEvents = append(pubEvents, events.ToMapStr())
		}
		beat.Events.PublishEvents(pubEvents, Publisher.Sync)

		fmt.Println("Events sent: ", len(events))

		// 告诉 registrar 我们已经成功的发送了这些事件信息， 没发送一次 event 就持久化一次 registrar
		// Tell the registrar that we've successfully sent these events
		fb.registrar.Channel <- events
	}
}
