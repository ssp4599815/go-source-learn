package cfgfile

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// Command line flags
var configfile *string


func Read(out interface{}, path string) error {
	if path == "" {
		path = *configfile
	}

	// 读取配置文件
	filecontent, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Failed to read %s :%v . Exiting.\n", path, err)
	}
	// 验证是否能够解析配置文件
	if err = yaml.Unmarshal(filecontent, out); err != nil {
		return fmt.Errorf("YAML config parsing failed on :%s %v. Exiting", path, err)
	}
	return nil
}
