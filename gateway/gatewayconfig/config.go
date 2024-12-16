package gatewayconfig

type ConfigGetter interface {
	GetGateway() Config
}

type Config struct {
	Addr            string `yaml:"addr"`
	Domain          string `yaml:"domain"`
	StaticFilesURL  string `yaml:"staticFilesUrl"`
	PublishFilesURL string `yaml:"publishFilesUrl"`
	ServeStatic     bool   `yaml:"serveStatic"`
	ServePublish    bool   `yaml:"servePublish"`
}
