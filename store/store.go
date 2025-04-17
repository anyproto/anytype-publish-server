package store

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"go.uber.org/zap"
)

var (
	ErrNotFound = errors.New("not found")
)

func New() Store {
	return &store{}
}

const CName = "publish.store"

var log = logger.NewNamed(CName)

type Store interface {
	app.Component

	Put(ctx context.Context, file File) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	DeletePath(ctx context.Context, path string) error
}

type store struct {
	bucket *string
	client *s3.Client
}

func (s *store) Init(a *app.App) (err error) {
	conf := a.MustComponent("config").(configSource).GetS3Store()
	if conf.Bucket == "" {
		return fmt.Errorf("s3 bucket is empty")
	}

	var awsConf aws.Config
	if conf.Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               "https://storage.googleapis.com",
				SigningRegion:     "auto",
				HostnameImmutable: true,
			}, nil
		})
		// TODO: handle env credentials
		if conf.Credentials.AccessKey != "" && conf.Credentials.SecretKey != "" {

		}

		awsConf, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(conf.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
			config.WithEndpointResolverWithOptions(customResolver),
		)

	} else {
		awsConf, err = config.LoadDefaultConfig(context.TODO())
		awsConf.Region = conf.Region
		if conf.Credentials.AccessKey != "" && conf.Credentials.SecretKey != "" {
			awsConf.Credentials = credentials.NewStaticCredentialsProvider(conf.Credentials.AccessKey, conf.Credentials.SecretKey, "")
		}

	}

	if err != nil {
		return err
	}

	// If creds are provided in the configuration, they are directly forwarded to the client as static credentials.

	s.bucket = aws.String(conf.Bucket)
	s.client = s3.NewFromConfig(awsConf)
	log.Info("s3 started", zap.String("region", conf.Region), zap.String("bucket", *s.bucket))
	return nil
}

func (s *store) Name() string {
	return CName
}

func (s *store) Put(ctx context.Context, file File) error {
	input := &s3.PutObjectInput{
		Bucket:        s.bucket,
		Key:           &file.Name,
		Body:          file.Reader,
		ContentType:   aws.String(file.ContentType()),
		ContentLength: aws.Int64(int64(file.Size)),
	}
	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		log.Warn("put s3", zap.String("key", file.Name), zap.Error(err))
		return err
	}
	return nil
}

func (s *store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: s.bucket,
		Key:    &key,
	}
	output, err := s.client.GetObject(ctx, input)
	if err != nil {
		var notFound *types.NoSuchKey
		if ok := errors.As(err, &notFound); ok {
			return nil, ErrNotFound
		} else {
			return nil, err
		}
	}
	return output.Body, nil
}

func (s *store) DeletePath(ctx context.Context, path string) error {
	output, err := s.client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: s.bucket,
		Prefix: &path,
	})
	if err != nil {
		return err
	}
	objects := make([]types.ObjectIdentifier, len(output.Contents))
	for i, c := range output.Contents {
		objects[i] = types.ObjectIdentifier{Key: c.Key}
	}
	input := &s3.DeleteObjectsInput{
		Bucket: s.bucket,
		Delete: &types.Delete{
			Objects: objects,
		},
	}
	_, err = s.client.DeleteObjects(ctx, input)
	if err != nil {
		return err
	}
	return nil
}
