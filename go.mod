module github.com/mysteriumnetwork/node

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Microsoft/go-winio v0.4.14
	github.com/andybalholm/brotli v1.0.0 // indirect
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be // indirect
	github.com/arthurkiller/rollingwriter v1.1.2
	github.com/asaskevich/EventBus v0.0.0-20180315140547-d46933a94f05
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/asdine/storm/v3 v3.1.1
	github.com/aws/aws-sdk-go-v2 v1.3.2
	github.com/aws/aws-sdk-go-v2/config v1.1.6
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.1.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.5.0
	github.com/cenkalti/backoff/v4 v4.0.0
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/denisenkom/go-mssqldb v0.0.0-20200620013148-b91950f658ec // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/ethereum/go-ethereum v1.10.2
	github.com/frankban/quicktest v1.5.0 // indirect
	github.com/gballet/go-libpcsclite v0.0.0-20191108122812-4678299bea08 // indirect
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-gonic/gin v1.7.2
	github.com/go-openapi/strfmt v0.19.3
	github.com/go-ozzo/ozzo-validation v3.6.0+incompatible
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2
	github.com/google/go-github/v35 v35.2.0
	github.com/huin/goupnp v1.0.1-0.20210310174557-0ca763054c88
	github.com/jackpal/gateway v1.0.6
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jinzhu/now v1.1.1 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/julienschmidt/httprouter v1.2.0
	github.com/karalabe/usb v0.0.0-20191104083709-911d15fe12a9 // indirect
	github.com/klauspost/pgzip v1.2.4 // indirect
	github.com/koron/go-ssdp v0.0.2
	github.com/kr/pretty v0.2.1 // indirect
	github.com/lib/pq v1.7.0 // indirect
	github.com/libp2p/go-libp2p v0.5.2
	github.com/libp2p/go-libp2p-core v0.3.0
	github.com/magefile/mage v1.11.0
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/miekg/dns v1.1.29
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/multiformats/go-multiaddr v0.2.0
	github.com/mysteriumnetwork/feedback v1.1.1
	github.com/mysteriumnetwork/go-ci v0.0.0-20200415074834-39fc864b0ed4
	github.com/mysteriumnetwork/go-dvpn-web v0.4.0
	github.com/mysteriumnetwork/go-openvpn v0.0.23
	github.com/mysteriumnetwork/go-wondershaper v1.0.1
	github.com/mysteriumnetwork/gowinlog v0.0.0-20200817095141-ad6c5f74d12e
	github.com/mysteriumnetwork/metrics v0.0.15
	github.com/mysteriumnetwork/payments v1.0.0
	github.com/nats-io/nats-server/v2 v2.3.2
	github.com/nats-io/nats.go v1.11.1-0.20210623165838-4b75fc59ae30
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/oleksandr/bonjour v0.0.0-20160508152359-5dcf00d8b228
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/oschwald/geoip2-golang v1.1.0
	github.com/oschwald/maxminddb-golang v1.5.0 // indirect
	github.com/pierrec/lz4 v2.5.2+incompatible // indirect
	github.com/pion/stun v0.3.5
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.17.2
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20200627165143-92b8a710ab6c
	github.com/songgao/water v0.0.0-20190112225332-f6122f5b2fbd
	github.com/spf13/cast v1.3.1
	github.com/status-im/keycard-go v0.0.0-20191114114615-9d48af884d5b // indirect
	github.com/stretchr/testify v1.7.0
	github.com/takama/daemon v1.0.0
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/urfave/cli/v2 v2.3.0
	github.com/vcraescu/go-paginator v0.0.0-20200304054438-86d84f27c0b3
	github.com/xtaci/kcp-go/v5 v5.5.8
	go.etcd.io/bbolt v1.3.4
	go.mongodb.org/mongo-driver v1.7.0
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
	golang.org/x/net v0.0.0-20210226172049-e18ecbb05110
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1
	golang.zx2c4.com/wireguard v0.0.20200320
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20200324154536-ceff61240acf
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/protobuf v1.26.0
	gopkg.in/src-d/go-git.v4 v4.13.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
