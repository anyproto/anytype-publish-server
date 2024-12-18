package store

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/anyproto/any-sync/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

func TestStore_Put(t *testing.T) {
	//t.Skip()
	fx := newFixture(t)
	data := bytes.NewReader([]byte("some data"))
	require.NoError(t, fx.Put(ctx, File{Name: "some/key", ContentSize: int(data.Size()), Reader: data}))
	reader, err := fx.Get(ctx, "some/key")
	require.NoError(t, err)
	result, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, "some data", string(result))
	require.NoError(t, fx.DeletePath(ctx, "some/"))
	_, err = fx.Get(ctx, "some/key")
	assert.ErrorIs(t, err, ErrNotFound)
}

type fixture struct {
	Store
	a *app.App
}

func newFixture(t *testing.T) *fixture {
	fx := &fixture{
		Store: New(),
		a:     new(app.App),
	}
	config := &testConfig{
		s3: Config{
			Region: "eu-central-1",
			Bucket: "anytype-gobackend-test",
		},
	}
	fx.a.Register(fx.Store).Register(config)
	require.NoError(t, fx.a.Start(ctx))
	t.Cleanup(func() {
		require.NoError(t, fx.a.Close(ctx))
	})
	return fx
}

type testConfig struct {
	s3 Config
}

func (t testConfig) Init(a *app.App) (err error) { return }
func (t testConfig) Name() (name string)         { return "config" }

func (t testConfig) GetS3Store() Config {
	return t.s3
}
