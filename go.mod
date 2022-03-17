module github.com/mysteriumnetwork/node

go 1.17

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/Microsoft/go-winio v0.5.0
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be
	github.com/arthurkiller/rollingwriter v1.1.3-0.20220211070658-c19a8e8b35be
	github.com/asaskevich/EventBus v0.0.0-20200907212545-49d423059eef
	github.com/asdine/storm/v3 v3.1.1
	github.com/aws/aws-sdk-go-v2 v1.3.2
	github.com/aws/aws-sdk-go-v2/config v1.1.6
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.1.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.5.0
	github.com/cenkalti/backoff/v4 v4.0.0
	github.com/chzyer/readline v0.0.0-20180603132655-2972be24d48e
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/ethereum/go-ethereum v1.10.9
	github.com/gin-contrib/cors v1.3.0
	github.com/gin-gonic/gin v1.7.2
	github.com/go-openapi/strfmt v0.19.3
	github.com/go-ozzo/ozzo-validation v3.6.0+incompatible
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/golang/protobuf v1.5.2
	github.com/google/go-github/v35 v35.2.0
	github.com/huin/goupnp v1.0.2
	github.com/jackpal/gateway v1.0.6
	github.com/julienschmidt/httprouter v1.2.0
	github.com/koron/go-ssdp v0.0.2
	github.com/libp2p/go-libp2p v0.5.2
	github.com/libp2p/go-libp2p-core v0.3.0
	github.com/magefile/mage v1.11.0
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/miekg/dns v1.1.29
	github.com/multiformats/go-multiaddr v0.2.0
	github.com/mysteriumnetwork/feedback v1.1.2-0.20211228095831-9dfca34c9ab7
	github.com/mysteriumnetwork/go-ci v0.0.0-20211124142828-37ca8ff3ef34
	github.com/mysteriumnetwork/go-dvpn-web v1.8.0
	github.com/mysteriumnetwork/go-openvpn v0.0.23
	github.com/mysteriumnetwork/go-wondershaper v1.0.1
	github.com/mysteriumnetwork/gowinlog v0.0.0-20200817095141-ad6c5f74d12e
	github.com/mysteriumnetwork/metrics v0.0.19
	github.com/mysteriumnetwork/payments v1.0.1-0.20211025073343-0b355972a602
	github.com/mysteriumnetwork/terms v0.0.40
	github.com/nats-io/nats.go v1.11.1-0.20210623165838-4b75fc59ae30
	github.com/oleksandr/bonjour v0.0.0-20160508152359-5dcf00d8b228
	github.com/oschwald/geoip2-golang v1.1.0
	github.com/pion/stun v0.3.5
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.26.1
	github.com/shurcooL/vfsgen v0.0.0-20200627165143-92b8a710ab6c
	github.com/songgao/water v0.0.0-20190112225332-f6122f5b2fbd
	github.com/spf13/cast v1.3.1
	github.com/stretchr/testify v1.7.0
	github.com/takama/daemon v1.0.0
	github.com/urfave/cli/v2 v2.3.0
	github.com/vcraescu/go-paginator v0.0.0-20200304054438-86d84f27c0b3
	github.com/xtaci/kcp-go/v5 v5.6.1
	go.etcd.io/bbolt v1.3.5
	go.mongodb.org/mongo-driver v1.7.0
	golang.org/x/crypto v0.0.0-20211215165025-cf75a172585e
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	golang.org/x/sys v0.0.0-20211214234402-4825e8c3871d
	golang.zx2c4.com/wireguard v0.0.0-20220117163742-e0b8f11489c5
	golang.zx2c4.com/wireguard/tun/netstack v0.0.0-20211017052713-f87e87af0d9a
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20211230205640-daad0b7ba671
	google.golang.org/protobuf v1.26.0
)

require (
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/andybalholm/brotli v1.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.1.6 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.0.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.0.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.0.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.2.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.1.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.3.0 // indirect
	github.com/aws/smithy-go v1.3.1 // indirect
	github.com/btcsuite/btcd v0.20.1-beta // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20200620013148-b91950f658ec // indirect
	github.com/dsnet/compress v0.0.1 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/erikstmartin/go-testdb v0.0.0-20160219214506-8d10e4a1bae5 // indirect
	github.com/gballet/go-libpcsclite v0.0.0-20191108122812-4678299bea08 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.1.0 // indirect
	github.com/go-git/go-git/v5 v5.3.0 // indirect
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/go-openapi/errors v0.19.2 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/go-github/v28 v28.1.1 // indirect
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/uuid v1.2.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/ipfs/go-cid v0.0.5 // indirect
	github.com/ipfs/go-ipfs-util v0.0.1 // indirect
	github.com/ipfs/go-log v0.0.1 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jbenet/go-temp-err-catcher v0.0.0-20150120210811-aac704a3f4f2 // indirect
	github.com/jbenet/goprocess v0.1.3 // indirect
	github.com/jinzhu/gorm v1.9.2 // indirect
	github.com/jinzhu/inflection v0.0.0-20180308033659-04140366298a // indirect
	github.com/jinzhu/now v1.1.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/native v0.0.0-20200817173448-b6b71def0850 // indirect
	github.com/json-iterator/go v1.1.11 // indirect
	github.com/kevinburke/ssh_config v1.1.0 // indirect
	github.com/klauspost/compress v1.11.13 // indirect
	github.com/klauspost/cpuid v1.3.1 // indirect
	github.com/klauspost/pgzip v1.2.4 // indirect
	github.com/klauspost/reedsolomon v1.9.9 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/lib/pq v1.7.0 // indirect
	github.com/libp2p/go-addr-util v0.0.1 // indirect
	github.com/libp2p/go-buffer-pool v0.0.2 // indirect
	github.com/libp2p/go-conn-security-multistream v0.1.0 // indirect
	github.com/libp2p/go-eventbus v0.1.0 // indirect
	github.com/libp2p/go-flow-metrics v0.0.3 // indirect
	github.com/libp2p/go-libp2p-autonat v0.1.1 // indirect
	github.com/libp2p/go-libp2p-circuit v0.1.4 // indirect
	github.com/libp2p/go-libp2p-discovery v0.2.0 // indirect
	github.com/libp2p/go-libp2p-loggables v0.1.0 // indirect
	github.com/libp2p/go-libp2p-mplex v0.2.1 // indirect
	github.com/libp2p/go-libp2p-nat v0.0.5 // indirect
	github.com/libp2p/go-libp2p-peerstore v0.1.4 // indirect
	github.com/libp2p/go-libp2p-secio v0.2.1 // indirect
	github.com/libp2p/go-libp2p-swarm v0.2.2 // indirect
	github.com/libp2p/go-libp2p-transport-upgrader v0.1.1 // indirect
	github.com/libp2p/go-libp2p-yamux v0.2.1 // indirect
	github.com/libp2p/go-maddr-filter v0.0.5 // indirect
	github.com/libp2p/go-mplex v0.1.0 // indirect
	github.com/libp2p/go-msgio v0.0.4 // indirect
	github.com/libp2p/go-nat v0.0.4 // indirect
	github.com/libp2p/go-openssl v0.0.4 // indirect
	github.com/libp2p/go-reuseport v0.0.1 // indirect
	github.com/libp2p/go-reuseport-transport v0.0.2 // indirect
	github.com/libp2p/go-stream-muxer-multistream v0.2.0 // indirect
	github.com/libp2p/go-tcp-transport v0.1.1 // indirect
	github.com/libp2p/go-ws-transport v0.2.0 // indirect
	github.com/libp2p/go-yamux v1.2.3 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/mdlayher/genetlink v1.1.0 // indirect
	github.com/mdlayher/netlink v1.4.2 // indirect
	github.com/mdlayher/socket v0.0.0-20211102153432-57e3fa563ecb // indirect
	github.com/mholt/archiver/v3 v3.3.0 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/minio/sha256-simd v0.1.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mmcloughlin/avo v0.0.0-20200803215136-443f81d77104 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/mr-tron/base58 v1.1.3 // indirect
	github.com/multiformats/go-base32 v0.0.3 // indirect
	github.com/multiformats/go-multiaddr-dns v0.2.0 // indirect
	github.com/multiformats/go-multiaddr-fmt v0.1.0 // indirect
	github.com/multiformats/go-multiaddr-net v0.1.2 // indirect
	github.com/multiformats/go-multibase v0.0.1 // indirect
	github.com/multiformats/go-multihash v0.0.13 // indirect
	github.com/multiformats/go-multistream v0.1.1 // indirect
	github.com/multiformats/go-varint v0.0.5 // indirect
	github.com/nats-io/nats-server/v2 v2.3.2 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/oschwald/maxminddb-golang v1.5.0 // indirect
	github.com/pierrec/lz4 v2.5.2+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rjeczalik/notify v0.9.2 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/spacemonkeygo/spacelog v0.0.0-20180420211403-2296661a0572 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/status-im/keycard-go v0.0.0-20191114114615-9d48af884d5b // indirect
	github.com/templexxx/cpu v0.0.7 // indirect
	github.com/templexxx/xorsimd v0.4.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tjfoc/gmsm v1.3.2 // indirect
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/whyrusleeping/go-logging v0.0.1 // indirect
	github.com/whyrusleeping/mafmt v1.2.8 // indirect
	github.com/whyrusleeping/multiaddr-filter v0.0.0-20160516205228-e903e4adabd7 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	go.opencensus.io v0.23.0 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	golang.org/x/tools v0.1.8 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	golang.zx2c4.com/go118/netip v0.0.0-20211111135330-a4a02eeacf9d // indirect
	golang.zx2c4.com/wintun v0.0.0-20211104114900-415007cec224 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	gvisor.dev/gvisor v0.0.0-20210506004418-fbfeba3024f0 // indirect
	honnef.co/go/tools v0.2.2 // indirect
)
