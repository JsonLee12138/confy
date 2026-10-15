// Package nacos adapts the official Nacos SDK to confy's Provider contracts.
package nacos

import (
	"github.com/JsonLee12138/confy/core"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

type Option func(*options)
type options struct {
	servers       []constant.ServerConfig
	client        constant.ClientConfig
	dataID, group string
	format        core.FileType
}

func WithServers(servers []constant.ServerConfig) Option {
	return func(o *options) { o.servers = append([]constant.ServerConfig(nil), servers...) }
}
func WithClientConfig(client constant.ClientConfig) Option {
	return func(o *options) { o.client = client }
}
func WithDataID(id string) Option            { return func(o *options) { o.dataID = id } }
func WithGroup(group string) Option          { return func(o *options) { o.group = group } }
func WithFormat(format core.FileType) Option { return func(o *options) { o.format = format } }
