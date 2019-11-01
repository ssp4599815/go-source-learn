package main

import (
	"fmt"
	"github.com/ssp4599815/go-source-learn/prometheus-v1.0.0/storage/local"
	"os"
)

func main() {
	os.Exit(Main())
}

func init() {
}

func Main() int {
	fmt.Println("Starting prometheus")

	var (
		memStorage = local.NewMemorySeriesStorage(cfg.storage)

	)
}
