package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_FetchConfigs(t *testing.T) {
	stat, _ := os.Stat("/tmp")

	fmt.Println(stat.Name())

	files, err := filepath.Glob( "/tmp/*.yaml")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(files)
	fmt.Println(filepath.Dir("/tmp/123/123"))
}

