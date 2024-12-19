package publish

type configGetter interface {
	GetPublish() Config
}

type Config struct {
	UploadUrlPrefix string `yaml:"uploadUrlPrefix"`
	HttpApiAddr     string `yaml:"httpApiAddr"`
	CleanupOn       bool   `yaml:"cleanupOn"`
}
