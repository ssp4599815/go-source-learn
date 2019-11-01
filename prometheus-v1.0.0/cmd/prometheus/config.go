package main

import (
	"github.com/ssp4599815/go-source-learn/prometheus-v1.0.0/storage/local"
)

var cfg = struct {
	printVersion bool
	configFile   string

	storage     local.MemorySeriesStorageOptions

}{
}

func init() {

}
