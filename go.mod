module github.com/mysteriumnetwork/node

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/andybalholm/brotli v1.0.0 // indirect
	github.com/arthurkiller/rollingwriter v1.1.2
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05
	github.com/asdine/storm/v3 v3.1.1
	github.com/awnumar/memguard v0.21.0
	github.com/aws/aws-sdk-go-v2 v0.15.0
	github.com/cenkalti/backoff/v4 v4.0.0
	github.com/cheggaaa/pb/v3 v3.0.1
	github.com/chzyer/logex v1.1.10 // indirect
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/chzyer/test v0.0.0-20180213035817-a1ea475d72b1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/ethereum/go-ethereum v1.9.6
	github.com/frankban/quicktest v1.5.0 // indirect
	github.com/gballet/go-libpcsclite v0.0.0-20191108122812-4678299bea08 // indirect
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-gonic/gin v1.4.0
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/golang/protobuf v1.4.0
	github.com/huin/goupnp v1.0.0
	github.com/jackpal/gateway v1.0.5
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/julienschmidt/httprouter v1.2.0
	github.com/karalabe/usb v0.0.0-20191104083709-911d15fe12a9 // indirect
	github.com/klauspost/compress v1.10.4 // indirect
	github.com/klauspost/pgzip v1.2.3 // indirect
	github.com/koron/go-ssdp v0.0.0-20180514024734-4a0ed625a78b
	github.com/kr/pretty v0.2.0 // indirect
	github.com/magefile/mage v1.9.0
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/miekg/dns v1.1.22
	github.com/mysteriumnetwork/feedback v1.1.1
	github.com/mysteriumnetwork/go-ci v0.0.0-20200316165146-af25c6390269
	github.com/mysteriumnetwork/go-dvpn-web v0.0.36
	github.com/mysteriumnetwork/go-openvpn v0.0.22
	github.com/mysteriumnetwork/go-wondershaper v1.0.1
	github.com/mysteriumnetwork/metrics v0.0.3
	github.com/mysteriumnetwork/payments v0.0.11
	github.com/nats-io/gnatsd v1.4.1 // indirect
	github.com/nats-io/go-nats v1.4.0
	github.com/nats-io/nats-server v1.4.1
	github.com/nats-io/nuid v1.0.1-0.20180712044959-3024a71c3cbe // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/oleksandr/bonjour v0.0.0-20160508152359-5dcf00d8b228
	github.com/olekukonko/tablewriter v0.0.2 // indirect
	github.com/onsi/ginkgo v1.11.0 // indirect
	github.com/onsi/gomega v1.8.1 // indirect
	github.com/oschwald/geoip2-golang v1.1.0
	github.com/oschwald/maxminddb-golang v1.5.0 // indirect
	github.com/pierrec/lz4 v2.5.1+incompatible // indirect
	github.com/pkg/errors v0.8.1
	github.com/robfig/cron v1.2.0 // indirect
	github.com/rs/zerolog v1.17.2
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/songgao/water v0.0.0-20190112225332-f6122f5b2fbd
	github.com/spf13/cast v1.3.0
	github.com/status-im/keycard-go v0.0.0-20191114114615-9d48af884d5b // indirect
	github.com/steakknife/bloomfilter v0.0.0-20180922174646-6819c0d2a570 // indirect
	github.com/steakknife/hamming v0.0.0-20180906055917-c99c65617cd3 // indirect
	github.com/stretchr/testify v1.4.1-0.20200130210847-518a1491c713
	github.com/ulikunitz/xz v0.5.7 // indirect
	github.com/urfave/cli/v2 v2.1.1
	github.com/xtaci/kcp-go/v5 v5.5.8
	golang.org/x/crypto v0.0.0-20200406173513-056763e48d71
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/sys v0.0.0-20200413165638-669c56c373c4
	golang.zx2c4.com/wireguard v0.0.20200320
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200324154536-ceff61240acf
	google.golang.org/protobuf v1.21.0
	gopkg.in/src-d/go-git.v4 v4.13.1 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

replace github.com/nats-io/go-nats v1.4.0 => github.com/mysteriumnetwork/nats.go v1.4.1-0.20200303115848-b4a5324c56ed
