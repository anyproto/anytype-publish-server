package config

import (
	"os"

	"github.com/anyproto/any-sync/app"
	"gopkg.in/yaml.v3"

	"github.com/anyproto/anytype-publish-server/db"
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
	Mongo   db.Mongo       `yaml:"mongo"`
	Publish publish.Config `yaml:"publish"`
	S3Store store.Config   `yaml:"s3Store"`
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
