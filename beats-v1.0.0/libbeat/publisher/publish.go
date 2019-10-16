package publisher

// 发货人
type ShipperConfig struct {
	Name                string
	RefreshTopologyFreq int // 刷新拓扑结构的频率
	IgnoreOutgoing      bool
	TopologyExpire      int // 拓扑结构的存活时间
	Tags                []string
	// Geoip 根据ip获取地址位置
}
