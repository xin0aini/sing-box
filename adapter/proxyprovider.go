package adapter

import (
	"github.com/sagernet/sing-box/option"
	"time"
)

type ProxyProvider interface {
	Tag() string
	Update() error
	ForceUpdate() error
	GetOutbounds() ([]Outbound, error)
	GetOutboundOptions() ([]option.Outbound, error)
	GetUpdateTime() time.Time
	GetSubscribeInfo() SubScribeInfo
}

type SubScribeInfo interface {
	GetUpload() uint64
	GetDownload() uint64
	GetTotal() uint64
	GetExpire() time.Time
}
