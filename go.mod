module github.com/anyproto/anytype-publish-server

go 1.23.8

toolchain go1.24.0

require (
	github.com/ahmetb/govvv v0.3.0
	github.com/anyproto/any-sync v0.9.2
	github.com/anyproto/anytype-publish-renderer v0.3.22
	github.com/anyproto/anytype-publish-server/publishclient v0.0.0-20250716122732-cdcfe3a126bb
	github.com/aws/aws-sdk-go-v2 v1.38.1
	github.com/aws/aws-sdk-go-v2/config v1.31.2
	github.com/aws/aws-sdk-go-v2/credentials v1.18.6
	github.com/aws/aws-sdk-go-v2/service/s3 v1.71.0
	github.com/aws/smithy-go v1.22.5
	github.com/golang/snappy v1.0.0
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.7.0
	github.com/stretchr/testify v1.10.0
	go.mongodb.org/mongo-driver v1.17.3
	go.uber.org/zap v1.27.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/a-h/templ v0.3.833 // indirect
	github.com/anyproto/any-store v0.3.3 // indirect
	github.com/anyproto/anytype-heart v0.42.0-rc30 // indirect
	github.com/anyproto/go-chash v0.1.0 // indirect
	github.com/anyproto/go-slip10 v1.0.0 // indirect
	github.com/anyproto/go-slip21 v1.0.0 // indirect
	github.com/anyproto/go-sqlite v1.4.2-any // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.7 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.25 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.4.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.28.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.33.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.38.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/btcsuite/btcd v0.24.2 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.5 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.6 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.1.0 // indirect
	github.com/btcsuite/btcutil v1.0.3-0.20220129005943-27c39e0ab4f9 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cheggaaa/mb/v3 v3.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/davidlazar/go-crypto v0.0.0-20200604182044-b73af7476f6c // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/ipfs/go-cid v0.5.0 // indirect
	github.com/ipfs/go-log/v2 v2.6.0 // indirect
	github.com/jbenet/go-temp-err-catcher v0.1.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/libp2p/go-libp2p v0.42.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mb0/diff v0.0.0-20131118162322-d8d9a906c24d // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multiaddr v0.16.0 // indirect
	github.com/multiformats/go-multibase v0.2.0 // indirect
	github.com/multiformats/go-multicodec v0.9.2 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-multistream v0.6.1 // indirect
	github.com/multiformats/go-varint v0.0.7 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20250313105119-ba97887b0a25 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.65.0 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/quic-go/quic-go v0.54.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/samber/lo v1.49.1 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/tyler-smith/go-bip39 v1.1.0 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/mock v0.5.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/exp v0.0.0-20250718183923-645b1fa84792 // indirect
	golang.org/x/mod v0.26.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/tools v0.35.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/Graylog2/go-gelf.v2 v2.0.0-20191017102106-1550ee647df0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	lukechampine.com/blake3 v1.4.1 // indirect
	modernc.org/libc v1.66.3 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.38.0 // indirect
	storj.io/drpc v0.0.34 // indirect
)

replace github.com/anyproto/anytype-publish-server/publishclient => ./publishclient

replace gopkg.in/Graylog2/go-gelf.v2 => github.com/anyproto/go-gelf v0.0.0-20210418191311-774bd5b016e7

replace github.com/btcsuite/btcutil => github.com/btcsuite/btcd/btcutil v1.1.5
