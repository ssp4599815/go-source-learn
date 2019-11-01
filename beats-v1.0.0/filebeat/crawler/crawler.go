package crawler

import (
	"fmt"
	"github.com/ssp4599815/beat/filebeat/config"
	"github.com/ssp4599815/beat/filebeat/input"
	"log"
	"os"
)

/*
 The hierarchy for the crawler objects is explained as following

 Crawler: Filebeat has one crawler（只有一个crawler）. The crawler is the single point of control
 	and stores the state. The state is written through the registrar
 Prospector: For every FileConfig the crawler starts a prospector (针对每个 FIleConfig 启动一个)
 Harvester: For every file found inside the FileConfig, the Prospector starts a Harvester (针对每个文件启动给一个)
 		The harvester send their events to the spooler
 		The spooler sends the event to the publisher
 		The publisher writes the state down with the registrar
*/

// 负责具体的日志收集工作
type Crawler struct {
	Registrar *Registrar // Registrar object to parsist the stat  持久化文件的状态信息
	running   bool       // 判断当前  crawer 是否正在运行，为后期 Stop() 操作留了一个 入口
}

// 启动一个 crawler 来抓取日志信息
func (crawler *Crawler) Start(files []config.ProspectorConfig, eventChan chan *input.FileEvent) {
	// 当前启动的 prospector个数
	pendingProspectorCnt := 0

	// Enable running
	crawler.running = true

	// 探测 所有的prospect中定义的日志文件，并为其 启动一个 harvester
	// Prospect the glob/paths given on the command line and launch harvesters
	for _, fileconfig := range files {
		fmt.Println("prospector", "File Configs: %v", fileconfig.Paths)

		// 初始化一个 Prospector
		prospector := &Prospector{
			ProspectorConfig: fileconfig,        // 这个是 每一个 要抓取的 日志文件信息
			registrar:        crawler.Registrar, // 每一个要持久化的 registrar 信息
		}

		// 对每一个要收集的日志文件，来初始化一个 prospector，并初始化配置信息
		err := prospector.Init()
		if err != nil {
			fmt.Printf("Error in initing prospector: %s", err)
			os.Exit(1)
		}

		// 每一个 prospector 启动一个 gorotion，并将读取到的日志 放到 eventChan 通道中，publisher会从 eventChan 中读取 fileevents
		go prospector.Run(eventChan)
		// 记录启动的 prospecter的个数
		pendingProspectorCnt++
	}

	// Now determine which states we need to persist by pulling the events from the prospectors
	// When we hit a nil source a prospector had finished so we decrease the expected events
	log.Printf("prospector, Waiting for %d prospector to initialise", pendingProspectorCnt)

	// 从通道中 获取要持久化的文件信息
	for event := range crawler.Registrar.Persist {
		// 如果日志文件为空
		if event.Source == nil {
			pendingProspectorCnt--
			// 如果要监听的日志文件为空，也就是没有要收集的日志了
			if pendingProspectorCnt == 0 {
				log.Printf("prospector, No pending prospectors.Finishing setup")
				break
			}
			continue
		}
		crawler.Registrar.State[*event.Source] = event
		log.Println("prospector, Registrar will re-save state for", *event.Source)

		// 如果 crawler 已经不再运行了，就退出
		if !crawler.running {
			break
		}
	}
}
func (crawler *Crawler) Stop() {
	// TODO: Properly stop prospectors and harvesters
}
