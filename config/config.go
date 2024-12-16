package config

import (
	"os"

	"github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/net/rpc"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
	"gopkg.in/yaml.v3"

	"github.com/anyproto/anytype-publish-server/db"
	"github.com/anyproto/anytype-publish-server/gateway/gatewayconfig"
	"github.com/anyproto/anytype-publish-server/publish"
	"github.com/anyproto/anytype-publish-server/store"
)

const CName = "config"

func NewFromFile(path string) (c *Config, err error) {
	c = &Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}
	return
}

type Config struct {
	Account                  accountservice.Config  `yaml:"account"`
	Mongo                    db.Mongo               `yaml:"mongo"`
	Publish                  publish.Config         `yaml:"publish"`
	S3Store                  store.Config           `yaml:"s3Store"`
	Drpc                     rpc.Config             `yaml:"drpc"`
	Yamux                    yamux.Config           `yaml:"yamux"`
	Quic                     quic.Config            `yaml:"quic"`
	Network                  nodeconf.Configuration `yaml:"network"`
	Gateway                  gatewayconfig.Config   `yaml:"gateway"`
	NetworkStorePath         string                 `yaml:"networkStorePath"`
	NetworkUpdateIntervalSec int                    `yaml:"networkUpdateIntervalSec"`
}

func (c *Config) Init(a *app.App) (err error) {
	return nil
}

func (c *Config) Name() (name string) {
	return CName
}

func (c *Config) GetMongo() db.Mongo {
	return c.Mongo
}

func (c *Config) GetPublish() publish.Config {
	return c.Publish
}

func (c *Config) GetS3Store() store.Config {
	return c.S3Store
}

func (c *Config) GetDrpc() rpc.Config {
	return c.Drpc
}

func (c *Config) GetAccount() accountservice.Config {
	return c.Account
}

func (c *Config) GetNodeConf() nodeconf.Configuration {
	return c.Network
}

func (c *Config) GetNodeConfStorePath() string {
	return c.NetworkStorePath
}

func (c *Config) GetNodeConfUpdateInterval() int {
	return c.NetworkUpdateIntervalSec
}

func (c *Config) GetYamux() yamux.Config {
	return c.Yamux
}

func (c *Config) GetQuic() quic.Config {
	return c.Quic
}

func (c *Config) GetSecureService() secureservice.Config {
	return secureservice.Config{RequireClientAuth: true}
}

func (c *Config) GetGateway() gatewayconfig.Config {
	return c.Gateway
}
