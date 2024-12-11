package publishclient

type configGetter interface {
	GetPublishServer() Config
}

type Config struct {
	Addrs []PublishServerAddr `yaml:"publishServerAddrs"`
}

type PublishServerAddr struct {
	PeerId string   `yaml:"peerId"`
	Addrs  []string `yaml:"addrs"`
}
