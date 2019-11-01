package publisher

import "github.com/ssp4599815/beat/libbeat/common"

// ClientOPtion allows API users to set additional options when publishing events
type ClientOption func(option *publishOptions)
type Client interface {
	PublishEvents(events []common.MapStr, opts ...ClientOption) bool
}

func Sync(options *publishOptions) {
	options.confirm = true
	options.sync = true
}
