package store

type configSource interface {
	GetS3Store() Config
}

type Credentials struct {
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
}

type Config struct {
	Endpoint    string      `yaml:"endpoint"`
	Region      string      `yaml:"region"`
	Bucket      string      `yaml:"bucket"`
	Credentials Credentials `yaml:"credentials"`
}
