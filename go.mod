module github.com/smartcontractkit/ocr2keepers

go 1.19

replace (
	github.com/btcsuite/btcd => github.com/btcsuite/btcd v0.23.3
	golang/github.com/gogo/protobuf => golang/github.com/gogo/protobuf v1.3.3
)

require (
	github.com/Maldris/mathparse v0.0.0-20170508133428-f0d009a7a773
	github.com/ethereum/go-ethereum v1.10.26
	github.com/go-echarts/go-echarts/v2 v2.2.5
	github.com/google/uuid v1.3.0
	github.com/pkg/errors v0.9.1
	github.com/shopspring/decimal v1.3.1
	github.com/smartcontractkit/libocr v0.0.0-20230531174957-6e75d6e613d1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.1
	go.uber.org/multierr v1.9.0
	golang.org/x/crypto v0.6.0
	gonum.org/v1/gonum v0.12.0
)

require (
	github.com/btcsuite/btcd/btcec/v2 v2.3.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.1.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rjeczalik/notify v0.9.3 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/tklauser/numcpus v0.6.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	golang.org/x/exp v0.0.0-20230213192124-5e25df0256eb // indirect
	golang.org/x/sys v0.5.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

exclude golang/github.com/influxdata/influxdb v1.8.3
