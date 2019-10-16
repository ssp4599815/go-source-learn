package outputs

// 定义日志输入时需要的一些基础信息
type MothershipConfig struct {
	SaveTopology      bool // 是否保存拓扑结构
	Host              string
	Port              int
	Hosts             []string
	LoadBalance       *bool
	Protocol          string
	Username          string
	Password          string
	Index             string
	Path              string
	Db                int
	DbTopology        int
	Timeout           int
	ReconnectInterval int
	Filename          string
	RotateEveryKb     int
	NumberOfFiles     int
	DataType          string
	FlushInterval     *int
	BulkMaxSize       *int `yaml:"bulk_max_size"`
	MaxRetries        *int
	Pretty            *bool
	Worker            int
}
