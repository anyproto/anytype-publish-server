package store

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
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

type RecalculateV4Signature struct {
	next   http.RoundTripper
	signer *v4.Signer
	cfg    aws.Config
}

func (lt *RecalculateV4Signature) RoundTrip(req *http.Request) (*http.Response, error) {
	// store for later use
	val := req.Header.Get("Accept-Encoding")

	// delete the header so the header doesn't account for in the signature
	req.Header.Del("Accept-Encoding")

	// sign with the same date
	timeString := req.Header.Get("X-Amz-Date")
	timeDate, _ := time.Parse("20060102T150405Z", timeString)

	creds, _ := lt.cfg.Credentials.Retrieve(req.Context())
	err := lt.signer.SignHTTP(req.Context(), creds, req, v4.GetPayloadHash(req.Context()), "s3", lt.cfg.Region, timeDate)
	if err != nil {
		return nil, err
	}
	// Reset Accept-Encoding if desired
	req.Header.Set("Accept-Encoding", val)

	fmt.Println("AfterAdjustment")
	rrr, _ := httputil.DumpRequest(req, false)
	fmt.Println(string(rrr))

	// follows up the original round tripper
	return lt.next.RoundTrip(req)
}

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
		log.Debug("loading custom S3 endpoint", zap.String("endpoint", conf.Endpoint))
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               conf.Endpoint,
				SigningRegion:     conf.Region,
				Source:            aws.EndpointSourceCustom,
				HostnameImmutable: true,
			}, nil
		})

		var opts []func(*config.LoadOptions) error
		opts = append(opts,
			config.WithRegion(conf.Region),
			config.WithEndpointResolverWithOptions(customResolver),
		)

		if conf.Credentials.AccessKey != "" && conf.Credentials.SecretKey != "" {
			log.Debug("loading custom S3 credentials", zap.String("ac", conf.Credentials.AccessKey))
			creds := credentials.NewStaticCredentialsProvider(conf.Credentials.AccessKey, conf.Credentials.SecretKey, "")
			opts = append(opts, config.WithCredentialsProvider(creds))
		}

		awsConf, err = config.LoadDefaultConfig(context.TODO(), opts...)
		awsConf.HTTPClient = &http.Client{Transport: &RecalculateV4Signature{http.DefaultTransport, v4.NewSigner(), awsConf}}

		fmt.Printf("awsconf: %#v\n", awsConf)
		fmt.Printf("conf: %#v\n", conf)
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
