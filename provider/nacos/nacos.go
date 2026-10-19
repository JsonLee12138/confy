package nacos

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/JsonLee12138/confy/core"
	"github.com/JsonLee12138/confy/provider"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/spf13/viper"
)

var (
	ErrOptions   = errors.New("confy/nacos: invalid options")
	ErrRead      = errors.New("confy/nacos: remote read failed")
	ErrDocument  = errors.New("confy/nacos: expected nonempty configuration object")
	ErrSubscribe = errors.New("confy/nacos: subscription failed")
)

// sdkClient deliberately excludes publish/delete and service discovery.
type sdkClient interface {
	GetConfig(vo.ConfigParam) (string, error)
	ListenConfig(vo.ConfigParam) error
	CancelListenConfig(vo.ConfigParam) error
	CloseClient()
}

// Provider owns its SDK client and private, ephemeral SDK cache directory.
// New prepares options and the shared Viper. Load or Watch initializes the SDK.
type Provider struct {
	loadMu    sync.Mutex
	configMu  sync.RWMutex
	v         *viper.Viper
	mu        sync.Mutex
	opts      options
	client    sdkClient
	directory string
	watcher   *watcher
	closed    bool
	factory   func(vo.NacosClientParam) (sdkClient, error)
}

var quietSDK sync.Once

// New creates a single-Data-ID Provider for one Confy Config.
// It defaults to YAML and DEFAULT_GROUP; validation and SDK initialization are deferred
// until Load or Watch.
func New(opts ...Option) *Provider {
	o := options{format: core.YAML, group: "DEFAULT_GROUP"}
	for _, opt := range opts {
		opt(&o)
	}
	if o.group == "" {
		o.group = "DEFAULT_GROUP"
	}
	v := viper.NewWithOptions(viper.WithDecoderRegistry(documentDecoders{viper.NewCodecRegistry()}))
	v.SetConfigType(string(o.format))
	return &Provider{opts: o, v: v, factory: func(param vo.NacosClientParam) (sdkClient, error) { return clients.NewConfigClient(param) }}
}

// resource identifies the Namespace/Group/Data ID with escaped path components and no credentials.
func (p *Provider) resource() string {
	return "nacos://" + url.PathEscape(p.opts.client.NamespaceId) + "/" + url.PathEscape(p.opts.group) + "/" + url.PathEscape(p.opts.dataID)
}

// safe attaches the source identity to a sentinel error without wrapping SDK errors or raw content.
func (p *Provider) safe(err error) error { return fmt.Errorf("%w: %s", err, p.resource()) }

// ConfigMutex synchronizes source updates with Confy reads and binding.
func (p *Provider) ConfigMutex() *sync.RWMutex { return &p.configMu }

// Close releases the owned subscription, client and ephemeral files.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	if p.watcher != nil {
		p.watcher.stopLocked()
	}
	if p.client != nil {
		p.client.CloseClient()
	}
	if p.directory != "" {
		if os.RemoveAll(p.directory) != nil {
			return p.safe(ErrRead)
		}
	}
	return nil
}

// Type identifies this adapter without making the consumer import the SDK.
func (*Provider) Type() provider.ProviderType { return provider.ProviderNacos }

var _ provider.Provider = (*Provider)(nil)
var _ provider.WatchProvider = (*Provider)(nil)
