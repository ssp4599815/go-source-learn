package service

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func HandleSignals(stopFunction func()) {
	var callback sync.Once

	// On ^C or SIGTERM, gracefully stop the sniffer
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	// 如果收到终止的信号，就执行 退出函数
	go func() {
		<-sigc
		callback.Do(stopFunction)
	}()
}
