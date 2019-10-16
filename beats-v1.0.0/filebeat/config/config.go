package config

import (
	"github.com/ssp4599815/beat/libbeat/cfgfile"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Default for config variables which are not set
const (
	DefaultRegistryFile                      = ".filebeat"
	DefaultSpoolSize           uint64        = 1024
	DefaultUdleTimeout         time.Duration = 5 * time.Second
	DefaultIgnoreOlderDuration time.Duration = 24 * time.Hour
	DefaultScanFrequency       time.Duration = 10 * time.Second
	DefaultHarvesterBufferSize int           = 16 << 10 // 16384
	DefaultInputType                         = "log"
	DefaultDocumentType                      = "log"
	DefaultBackoff                           = 1 * time.Second
	DefaultBackoffFactor                     = 2
	DefaultMaxBackoff                        = 10 * time.Second
	DefaultPartialLineWaiting                = 5 * time.Second
	DefaultForceCloseFiles                   = false
)

type Config struct {
	Filebeat FilebeatConfig
}

type FilebeatConfig struct {
	Prospectors         []ProspectorConfig            // 定义多个探测者
	SpoolSize           uint64 `yaml:"spool_size"`    // 线程池大小
	IdleTimeout         string `yaml:"idle_timeout"`  // 空闲的超时时间
	IdleTimeoutDuration time.Duration                 // 空闲的超时时间
	RegistryFile        string `yaml:"registry_file"` // 记录日志读取信息的文件
	ConfigDir           string `yaml:"config_dir"`    // 配置文件的位置
}

// 定义探测者
type ProspectorConfig struct {
	Paths                 []string                         // 要监听的所有的日志文件的路径
	Input                 string                           // 输入
	IgnoreOlder           string `yaml:"ignore_older"`     // 忽略多久的旧数据
	IgnoreOlderDruation   time.Duration                    // 忽略多久的旧数据
	ScanFrequency         string `yaml:"scan_frequency"`   // 间隔多久来读取一次日志
	ScanFrequencyDuration time.Duration                    // 间隔多久来读取一次日志
	Harvester             HarvesterConfig `yaml:",inline"` // 每一个读取日志的角色
}

// Harvester 真正读取日志的线程
type HarvesterConfig struct {
	InputType       string `yaml:"input_type"`            // 要读取的日志类型
	Fields          map[string]string                     // 自定义的 标签
	FieldsUnderRoot bool   `yaml:"fields_under_root"`     // 自定义标签是否在根
	BufferSize      int    `yaml:"harvester_buffer_size"` // 定义缓冲区大小
	TailFiles       bool   `yaml:"tail_files"`            // 是否持续读取追加的日志
	Encoding        string `yaml:"encoding"`              // 日志编码格式  utf-8 gbk plain ...
	DocumentType    string `yaml:"ducoment_type"`         // 日志的类型
	// backoff选项定义到达EOF后Filebeat在再次检查文件之前等待的时间.
	// 默认值为1s，这意味着如果添加了新行，则每秒检查一次文件. 这可以实现近实时抓取日志.
	// 每当文件中出现新行时， backoff值将重置为初始值. 默认值为1s.
	Backoff         string `yaml:"backoff"` // Filebeat检测到某个文件到了EOF（文件结尾）之后，每次等待多久再去检测文件是否有更新，默认为1s
	BackoffDuration time.Duration
	// 此选项指定增加等待时间的速度. 退避因子越大， max_backoff值越快达到. 退避因子呈指数增加.
	// 允许的最小值是1.如果将此值设置为1，则会禁用退避算法，并且backoff值用于等待新行. 每次将backoff值乘以backoff_factor直到达到max_backoff . 预设值为2.
	BackoffFactor int `yaml:"backoff_factor"`
	// 达到EOF后Filebeat等待再次检查文件的最长时间. 从检查文件中max_backoff无论为backoff_factor指定什么，等待时间都不会超过backoff_factor .
	// 因为读取新行最多需要10s， 为max_backoff指定10s意味着在最坏的情况下，如果Filebeat已多次退出，则可以在日志文件中添加新行. 默认值为10秒.
	MaxBackoff        string `yaml:"max_backoff"`
	MaxBackoffDurtion time.Duration
	// 有时Filebeat在完全写入之前先检查一行. 此选项指定harvester在跳过一行之前等待系统完成一行的时间. 默认值为5秒.
	PartialLineWating         string `yaml:"partial_line_wating"`
	PartialLineWatingDuration time.Duration
	// 默认情况下，Filebeat会将其读取的文件保持打开状态，直到经过ignore_older指定的时间跨度.
	// 删除文件时，此行为可能导致问题. 在Windows上，除非Filebeat关闭文件，否则无法完全删除该文件. 此外，在此期间无法创建具有相同名称的新文件.
	ForceCloseFiles bool `yaml:"force_close_files"`
}

// 返回要查看的配置文件，
// 如果路径是一个文件，则直接返回
// 如果路径是一个目录，则返回该目录下面所有 *.yml 文件
// getConfigFiles returns list  of config files
// In case path is a file, it will be directly returned
// In case it is a directory, it will fetch all .yml files inside this directory
func getConfigFiles(path string) (configFiles []string, err error) {
	// check if path is valid file or dir
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// create empty slice for config file list
	configFiles = make([]string, 0)

	// 入股配置中是个 目录，返回所有的 *.yml 配置文件
	if stat.IsDir() {
		// Glob函数返回所有匹配模式匹配字符串pattern的文件或者nil（如果没有匹配的文件）
		// func Glob(pattern string) (matches []string, err error)
		// 返回的是一个 文件绝对路径的列表
		files, err := filepath.Glob(path + "/*.yml")
		if err != nil {
			return nil, err
		}

		configFiles = append(configFiles, files...)
	} else {
		// only 1 config file
		configFiles = append(configFiles, path)
	}
	return configFiles, nil
}

// mergeConfigFiles reads in all config files given by list configFiles and merges them into config
// 合并所有的配置文件中的 Prospectors
func mergeConfigFiles(configFiles []string, config *Config) error {
	// 读取所有配置文件
	for _, file := range configFiles {
		tmpConfig := &Config{}

		// 解析 yaml 文件
		cfgfile.Read(tmpConfig, file)

		// 将所有的 Prospectors 整合到一起
		config.Filebeat.Prospectors = append(config.Filebeat.Prospectors, tmpConfig.Filebeat.Prospectors...)
	}
	return nil
}

// 将所有的配置文件整合一起
// Fetches and merges all config files given by configDir. All are put into one config object
func (config *Config) FetchConfigs() {
	// 配置文件的路径
	configDir := config.Filebeat.ConfigDir

	// If option not set, do nothing
	if configDir == "" {
		return
	}

	// 获取配置文件
	configFiles, err := getConfigFiles(configDir)
	if err != nil {
		log.Fatal("Colud not use config_dir of :", configDir, err)
	}

	// 整合配置文件
	err = mergeConfigFiles(configFiles, config)

	if err != nil {
		log.Fatal("Error merging config files: ", err)
	}
	if len(config.Filebeat.Prospectors) == 0 {
		log.Fatalf("No paths given, What files do you want me to watch?")
	}
}
