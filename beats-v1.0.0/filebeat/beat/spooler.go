package beat

import (
	"fmt"
	cfg "github.com/ssp4599815/beat/filebeat/config"
	"github.com/ssp4599815/beat/filebeat/input"
	"time"
)

// Spooler 负责从 harvester 中获取信息 然后 发送给 publisher
type Spooler struct {
	Filebeat      *Filebeat             // 将filebeat的相关信息传递进来
	running       bool                  // 是否正在运行
	nextFlushTime time.Time             // 每次的刷新的间隔时间
	spool         []*input.FileEvent    // spool 用来存储 日志信息
	Channel       chan *input.FileEvent // 用来接收日志信息的通道
}

// 初始化一个 spooler
func NewSpooler(filebeat *Filebeat) *Spooler {
	spooler := &Spooler{
		Filebeat: filebeat,
		running:  false,
	}
	// 获取配置相关信息
	config := &spooler.Filebeat.FbConfig.Filebeat

	// 每隔多长时间刷新一次 spool, 然后将 spool 里面的数据 发送到 publisher
	// Set the next flush time
	spooler.nextFlushTime = time.Now().Add(config.IdleTimeoutDuration)
	spooler.Channel = make(chan *input.FileEvent, 16)

	return spooler
}

// 对 spooler 进行配置
func (s *Spooler) Config() error {
	// 获取 filebeat 的配置文件
	config := &s.Filebeat.FbConfig.Filebeat

	// 设置缓冲区的大小，默认为 1024
	// set default pool size if value not set
	if config.SpoolSize == 0 {
		config.SpoolSize = cfg.DefaultSpoolSize
	}

	// 设置空闲的时间，默认为 5秒
	// set default idle timeout if not set
	if config.IdleTimeout == "" {
		fmt.Printf("Set idleTimeoutDuration to %s\n", cfg.DefaultUdleTimeout)
		// set it to default
		config.IdleTimeoutDuration = cfg.DefaultUdleTimeout
	} else {
		var err error
		config.IdleTimeoutDuration, err = time.ParseDuration(config.IdleTimeout)

		if err != nil {
			fmt.Printf("Failed to parse idle timeout duration '%s'. Error was: %v", config.IdleTimeout, err)
			return err
		}
	}
	return nil
}

// 启动 spooler， 并周期性的进行 心跳检测。
// 如果 最近一次刷新时间 超过了 IdleTimeoutDuration ，那么我们就进行一次 强制刷新，防止 spool 夯住。
// Run runs the spooler
// It heartbeats periodically. if the last flush was longer than
// 'IdleTimeoutDuration' time ago ,then we'll force a flush to prevent us from
// holding on to spooled events for too long
func (s *Spooler) Run() {
	// 获取配置信息
	config := &s.Filebeat.FbConfig.Filebeat

	// 开始运行 spooler
	// Enable running
	s.running = true

	// Sets up ticket channel   设定一个循环的定时器 ,为 IdleTimeoutDuration 的一半
	ticker := time.NewTicker(config.IdleTimeoutDuration / 2)

	// 初始化一个 spool 用来 存放 从通道中获取的 日志文件信息
	s.spool = make([]*input.FileEvent, 0, config.SpoolSize)

	fmt.Printf("starting spooler: spool_size :%v, idle_timeout: %s", config.SpoolSize, config.IdleTimeoutDuration)

	// Loops until running is set to false
	for {
		// 是为后续的 退出操作 做的一个准备
		if !s.running {
			break
		}

		select {
		// 从通道中获取 日志信息
		case event := <-s.Channel:
			s.spool = append(s.spool, event)

			// Spooler if full -> flush  ， 通道满了 就发送
			if len(s.spool) == cap(s.spool) {
				fmt.Printf("Flushing spooler because spooler full, Events flushed: %v", len(s.spool))
				// 执行刷新操作
				s.flush()
			}
		case <-ticker.C: // 周期性的检查
			// Flush periodically 周期性的进行刷新
			if time.Now().After(s.nextFlushTime) {
				fmt.Printf("Flush spooler because of timeout, Events flushed: %v", len(s.spool))
				s.flush()
			}
		}
	}

	fmt.Println("Stopping spooler")

	// 退出之前也执行一次刷新操作
	// Flush again before exiting spooler and closes channel
	s.flush()
	close(s.Channel)
}

// Stop stops the spooler. Flushes events before stopping
func (s *Spooler) Stop() {
}

// 将 spooler 中的 日志事件信息 发送到 publisher 通道中
func (s *Spooler) flush() {
	// 如果 有要发送的数据
	// Checks if any new object
	if len(s.spool) > 0 {
		// copy buffer
		tmpCopy := make([]*input.FileEvent, len(s.spool))
		copy(tmpCopy, s.spool)

		// clear buffer， 每次刷新都要清空 buffer
		s.spool = s.spool[:0]

		// 发送数据给 publisher 通道
		s.Filebeat.publisherChan <- tmpCopy
	}
	// 然后 将 下次刷新事件 增加！
	s.nextFlushTime = time.Now().Add(s.Filebeat.FbConfig.Filebeat.IdleTimeoutDuration)
}
