// Package nacos adapts the official Nacos SDK to confy's Provider contracts.
package nacos

import (
	"github.com/JsonLee12138/confy/core"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

// Option configures one Nacos Provider before loading initializes the SDK.
type Option func(*options)
type options struct {
	servers       []constant.ServerConfig
	client        constant.ClientConfig
	dataID, group string
	format        core.FileType
}

// WithServers copies the server list; an Endpoint may be supplied instead through WithClientConfig.
func WithServers(servers []constant.ServerConfig) Option {
	return func(o *options) { o.servers = append([]constant.ServerConfig(nil), servers...) }
}

// WithClientConfig supplies SDK connection, namespace, credentials, and timeout settings.
// Load replaces cache/log paths with private owned paths, disables snapshot fallback,
// and rejects OpenKMS and TLS TrustAll. CacheDir is the parent of the private cache;
// a zero TimeoutMs uses 5000 milliseconds.
func WithClientConfig(client constant.ClientConfig) Option {
	return func(o *options) { o.client = client }
}

// WithDataID selects the required remote document identifier; its extension does not choose the parser.
func WithDataID(id string) Option { return func(o *options) { o.dataID = id } }

// WithGroup selects a configuration group; an empty value uses DEFAULT_GROUP.
func WithGroup(group string) Option { return func(o *options) { o.group = group } }

// WithFormat selects YAML, JSON, or TOML decoding; New defaults to YAML.
func WithFormat(format core.FileType) Option { return func(o *options) { o.format = format } }
