module github.com/wormhole-foundation/wormhole-explorer/jobs

go 1.19

require (
	github.com/go-redis/redis v6.15.9+incompatible
	github.com/go-resty/resty/v2 v2.11.0
	github.com/google/uuid v1.3.0
	github.com/influxdata/influxdb-client-go/v2 v2.12.2
	github.com/pkg/errors v0.9.1
	github.com/shopspring/decimal v1.3.1
	github.com/stretchr/testify v1.8.4
	github.com/test-go/testify v1.1.4
	github.com/wormhole-foundation/wormhole-explorer/common v0.0.0-20230713181709-0425a89e7533
	github.com/wormhole-foundation/wormhole/sdk v0.0.0-20240416174455-25e60611a867
	go.mongodb.org/mongo-driver v1.11.2
	go.uber.org/zap v1.26.0
)

require (
	github.com/algorand/go-algorand-sdk v1.23.0 // indirect
	github.com/algorand/go-codec/codec v1.1.8 // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.2 // indirect
	github.com/certusone/wormhole/node v0.0.0-20240416174455-25e60611a867 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cosmos/btcutil v1.0.5 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/deepmap/oapi-codegen v1.8.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/ethereum/go-ethereum v1.11.3 // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
	github.com/gofiber/fiber/v2 v2.47.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/holiman/uint256 v1.2.1 // indirect
	github.com/influxdata/line-protocol v0.0.0-20210311194329-9aa0e372d097 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/montanaflynn/stats v0.7.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/philhofer/fwd v1.1.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/savsgio/dictpool v0.0.0-20221023140959-7bf2e61cea94 // indirect
	github.com/savsgio/gotils v0.0.0-20230208104028-c358bd845dee // indirect
	github.com/sethvargo/go-envconfig v1.0.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/tinylib/msgp v1.1.8 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.47.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.19.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sync v0.4.0 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/wormhole-foundation/wormhole-explorer/common => ../common
