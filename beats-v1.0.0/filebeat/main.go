package main

import (
	"fmt"
	filebeat "github.com/ssp4599815/beat/filebeat/beat"
	"github.com/ssp4599815/beat/libbeat/beat"
	"log"
)

var Version = "1.0.0"
var Name = "filebeat"

func main() {

	fmt.Println("开始启动 filebeat")
	// 创建一个 beat 对象
	fb := &filebeat.Filebeat{}

	// Initi bead objectfile
	b := beat.NewBeat(Name, Version, fb)

	// Loads base config 加载基础的配置文件
	b.LoadConfig()

	// 读取配置文件
	err := fb.Config(b)
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	// Run beat. this calls first beater.Setup,
	// then beater.Run and Beater.Cleanup in the end
	b.Run()
}
