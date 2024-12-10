package store

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/anyproto/any-sync/app"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	ErrNotFound = errors.New("not found")
)

func New() Store {
	return &store{}
}

const CName = "store"

type Store interface {
	app.Component

	Put(ctx context.Context, key string, reader io.Reader) error
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

	awsConf, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	// If creds are provided in the configuration, they are directly forwarded to the client as static credentials.
	if conf.Credentials.AccessKey != "" && conf.Credentials.SecretKey != "" {
		awsConf.Credentials = credentials.NewStaticCredentialsProvider(conf.Credentials.AccessKey, conf.Credentials.SecretKey, "")
	}
	awsConf.Region = conf.Region
	s.bucket = aws.String(conf.Bucket)
	s.client = s3.NewFromConfig(awsConf)
	return nil
}

func (s *store) Name() string {
	return CName
}

func (s *store) Put(ctx context.Context, key string, reader io.Reader) error {
	input := &s3.PutObjectInput{
		Bucket: s.bucket,
		Key:    &key,
		Body:   reader,
	}
	_, err := s.client.PutObject(ctx, input)
	if err != nil {
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
