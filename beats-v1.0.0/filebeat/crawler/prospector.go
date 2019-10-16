package crawler

import (
	"fmt"
	cfg "github.com/ssp4599815/beat/filebeat/config"
	"github.com/ssp4599815/beat/filebeat/harvester"
	"github.com/ssp4599815/beat/filebeat/input"
	"os"
	"path/filepath"
	"time"
)

// 探测所有的文件
type Prospector struct {
	ProspectorConfig cfg.ProspectorConfig
	prospectorList   map[string]prospectorFileStat // 所有的 prospector 列表
	iteration        uint32
	lastscan         time.Time              // 最后一次监听文件的时间
	registrar        *Registrar             // 要持久化的文件信息
	missingFiles     map[string]os.FileInfo // 要忽略的文件
	running          bool                   // prospector是否运行的标志位，用于后续 Stop()操作
}

type prospectorFileStat struct {
	Fileinfo os.FileInfo // the file info  prospector监听的文件
	// 关闭 harvester的时候 获取当期那文件的偏移量
	Harvester chan int64 // the harvester will send an event with its offset when it closes
	// 获取文件的最后一次的迭代的 号码
	LastIteration uint32 // int number of the last iteration in which we saw this file
}

// 根据默认的配置来初始化一个 prospector
// Init sets up default config for prospector
func (p *Prospector) Init() error {
	// 配置 prospecter用到的信息
	err := p.setupProspectorConfig()
	if err != nil {
		return err
	}

	// 配置harvester用到的信息
	err = p.setupHarvesterConfig()
	if err != nil {
		return err
	}
	return nil
}

// 设置 prospector 配置
// Setup Prospector config
func (p *Prospector) setupProspectorConfig() error {
	var err error
	config := &p.ProspectorConfig

	// 设置忽略的旧日志，默认时间为 24小时
	config.IgnoreOlderDruation, err = getConfigDuration(config.IgnoreOlder, cfg.DefaultIgnoreOlderDuration, "ignore_older")
	if err != nil {
		return err
	}

	// 设置检测文件变动的时间间隔，默认为 10秒
	config.ScanFrequencyDuration, err = getConfigDuration(config.ScanFrequency, cfg.DefaultScanFrequency, "scan_frequency")
	if err != nil {
		return err
	}

	// Init file stat list
	p.prospectorList = make(map[string]prospectorFileStat)
	return nil
}

// Setup Harvester Config
func (p *Prospector) setupHarvesterConfig() error {

	var err error
	config := &p.ProspectorConfig.Harvester

	// 设置 harvester 缓冲区的大小，默认为 16384
	// Setup Buffer Size
	if config.BufferSize == 0 {
		config.BufferSize = cfg.DefaultHarvesterBufferSize
	}

	// 设置日志类型，默认为 "log" 类型
	// Setup DocumentType
	if config.DocumentType == "" {
		config.DocumentType = cfg.DefaultDocumentType
	}

	// 设置 输入文件的类型，默认为 "log"
	// Setup InputType
	if config.InputType == "" {
		config.InputType = cfg.DefaultInputType
	}

	// 指定Filebeat如何积极地抓取新文件进行更新。默认1s
	// 定义Filebeat在达到EOF之后再次检查文件之间等待的时间。
	config.BackoffDuration, err = getConfigDuration(config.Backoff, cfg.DefaultBackoff, "backoff")
	if err != nil {
		return err
	}

	// 指定backoff尝试等待时间是几次，默认是2
	// Setup BackoffFactor
	if config.BackoffFactor == 0 {
		config.BackoffFactor = cfg.DefaultBackoffFactor
	}

	// 在达到EOF之后再次检查文件之前Filebeat等待的最长时间, 默认 10s
	config.MaxBackoffDurtion, err = getConfigDuration(config.MaxBackoff, cfg.DefaultMaxBackoff, "max_backoff")
	if err != nil {
		return err
	}
	return nil
}

// 开始监听所有的文件路径，并获取相关的日志文件。 每个文件启动一个 harvester
// Starts scanning through all the file paths and fetch the related files. start a harvester for each file
func (p *Prospector) Run(spoolChan chan *input.FileEvent) {
	// 开启 prospect ，方便后续 Stop() 操作
	p.running = true

	// Handle any "-" (stdin) paths ，处理任何文件，包括 标准输入
	for i, path := range p.ProspectorConfig.Paths { // 遍历所有的 日志路径信息 path
		fmt.Println("prospector Harvest path: ", path)

		// 如果 是一个 标准输入
		if path == "-" {
			// Offset and Initial never get used when path is "-"
			h, err := harvester.NewHarvester(p.ProspectorConfig, &p.ProspectorConfig.Harvester, path, nil, spoolChan)
			if err != nil {
				fmt.Println("Error initializing harvester: ", err)
				return
			}

			// 开启一个 goroutine 进行日志收集
			h.Start()

			// Remove it from the file list
			p.ProspectorConfig.Paths = append(p.ProspectorConfig.Paths[:i], p.ProspectorConfig.Paths[i+1:]...)
		}
	}

	// Seed last scan time
	p.lastscan = time.Now()

	// Now let's do one quick scan to pick up new files
	for _, path := range p.ProspectorConfig.Paths {
		p.scan(path, spoolChan)
	}
}

// Scans the specific path which can be a glob (/**/**/*.log)
// For all found files it is checked if a harvester should be started
func (p *Prospector) scan(path string, output chan *input.FileEvent) {
	fmt.Println("prospector,scan path ", path)
	// Evaluate（评估） the path as a wildcards(通配符)/shell glob
	matches, err := filepath.Glob(path)
	if err != nil {
		fmt.Printf("prospector, glob(%s) failed: %v", path, err)
		return
	}
	p.missingFiles = map[string]os.FileInfo{}

	// check any matched files to see if we need to start a harvester
	for _, file := range matches {
		fmt.Println("prospector, Check file for harvesting: ", file)

		// Stat the file, following any symlinks
		fileinfo, err := os.Stat(file)
		if err != nil {
			fmt.Printf("prospector, stat(%s) failed: %s", file, err)
			continue
		}

		newFile := input.File{
			FileInfo: fileinfo,
		}

		// 跳过目录文件
		if newFile.FileInfo.IsDir() {
			fmt.Println("prospector, Skipping directory: ", file)
			continue
		}

		// Check the current info against p.prospectorinfo[file]
		lastinfo, isKnown := p.prospectorList[file]

		oldFile := input.File{
			FileInfo: lastinfo.Fileinfo,
		}

		// Create a new prospector info with the stat info for comparison
		newInfo := prospectorFileStat{
			Fileinfo:      newFile.FileInfo,
			Harvester:     make(chan int64, 1),
			LastIteration: p.iteration,
		}

		// Conditions for starting a new harvester:
		// - file path hasn't been seen before
		// - the file's inode or device changed
		if !isKnown {
			p.checkNewFile(&newInfo, file, output)
		} else {
			newInfo.Harvester = lastinfo.Harvester
			p.checkExistingFile(&newInfo, &newFile, &oldFile, file, output)
		}
		// Track the stat data for this file for laster comparison to check for
		// rotation/etc
		p.prospectorList[file] = newInfo
	} // for each file methed by the glob
}

func (p *Prospector) checkNewFile(newinfo *prospectorFileStat, file string, output chan *input.FileEvent) {
	fmt.Println("prospector, Start harvesting unknown file: ", file)

	// Init harvester with info
	h, err := harvester.NewHarvester(
		p.ProspectorConfig, &p.ProspectorConfig.Harvester,
		file, newinfo.Harvester, output)
	if err != nil {
		fmt.Println("Error initializing harvester: ", err)
		return
	}

	// Chech for unmodified time, but only if the file modification time is before the last scan started
	// This ensures we don't skip genuine creations with dead times lass than 10s

	if newinfo.Fileinfo.ModTime().Before(p.lastscan) && time.Since(newinfo.Fileinfo.ModTime()) > p.ProspectorConfig.IgnoreOlderDruation {
		fmt.Println("prospector, Fetching old state of file to resume: ", file)

		offset, resuming := p.registrar.fetchState(file, newinfo.Fileinfo)
	}
}

// 解析  string 类型的 duration
// getConfigDuration builds the duration based on the input string
// returns error if an invalid string duration is passed
// In case no duration is set ,default duration will be used
func getConfigDuration(config string, duration time.Duration, name string) (time.Duration, error) {

	// Setup Ignore Older
	if config != "" {
		var err error
		duration, err = time.ParseDuration(config)
		if err != nil {
			fmt.Printf("Failed to parse %s value '%s'. Error was： %s\n", name, config, err)
			return 0, err
		}
	}
	fmt.Printf("prospector  Set %s duration to %s", name, duration)
	return duration, nil
}
