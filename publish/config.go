package publish

type configGetter interface {
	GetPublish() Config
}

type Config struct {
	UploadUrlPrefix string `yml:"uploadUrlPrefix"`
}
