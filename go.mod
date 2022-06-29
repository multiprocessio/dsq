module github.com/multiprocessio/dsq

go 1.18

// Uncomment for local development (and re-run `go mod tidy`)
// replace github.com/multiprocessio/datastation/runner => ../datastation/runner

require (
	github.com/chzyer/readline v1.5.0
	github.com/google/uuid v1.3.0
	github.com/multiprocessio/datastation/runner v0.0.0-20220628000637-8472e7fd559e
	github.com/olekukonko/tablewriter v0.0.5
)

require (
	cloud.google.com/go v0.100.2 // indirect
	cloud.google.com/go/bigquery v1.31.0 // indirect
	cloud.google.com/go/compute v1.5.0 // indirect
	cloud.google.com/go/iam v0.3.0 // indirect
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/azure-storage-blob-go v0.14.0 // indirect
	github.com/ClickHouse/clickhouse-go/v2 v2.0.14 // indirect
	github.com/alexbrainman/odbc v0.0.0-20211220213544-9c9a2e61c5e2 // indirect
	github.com/apache/arrow/go/arrow v0.0.0-20211112161151-bc219186db40 // indirect
	github.com/apache/thrift v0.15.0 // indirect
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de // indirect
	github.com/aws/aws-sdk-go v1.44.14 // indirect
	github.com/aws/aws-sdk-go-v2 v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.1.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.1.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.6.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.6.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.10.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.23.0 // indirect
	github.com/aws/smithy-go v1.9.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/deepmap/oapi-codegen v1.8.2 // indirect
	github.com/denisenkom/go-mssqldb v0.12.0 // indirect
	github.com/flosch/pongo2 v0.0.0-20200913210552-0d938eb266f3 // indirect
	github.com/form3tech-oss/jwt-go v3.2.5+incompatible // indirect
	github.com/gabriel-vasile/mimetype v1.4.0 // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/goccy/go-json v0.9.7 // indirect
	github.com/gocql/gocql v1.1.0 // indirect
	github.com/golang-sql/civil v0.0.0-20190719163853-cb61b32ac6fe // indirect
	github.com/golang-sql/sqlexp v0.0.0-20170517235910-f1bb20e5a188 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v2.0.5+incompatible // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/googleapis/gax-go/v2 v2.3.0 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/influxdata/influxdb-client-go/v2 v2.8.2 // indirect
	github.com/influxdata/line-protocol v0.0.0-20200327222509-2487e7298839 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jmoiron/sqlx v1.3.5 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/lib/pq v1.10.5 // indirect
	github.com/linkedin/goavro/v2 v2.11.1 // indirect
	github.com/matoous/go-nanoid/v2 v2.0.0 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.10 // indirect
	github.com/mattn/go-sqlite3 v1.14.14-0.20220530010643-3ccccfb4c9c6 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/multiprocessio/go-json v0.0.0-20220308002443-61d497dd7b9e // indirect
	github.com/multiprocessio/go-openoffice v0.0.0-20220110232726-064f5dda1956 // indirect
	github.com/multiprocessio/go-sqlite3-stdlib v0.0.0-20220601025455-7a933dfc26ed // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/neo4j/neo4j-go-driver/v4 v4.4.2 // indirect
	github.com/paulmach/orb v0.5.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.14 // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.12.2 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.34.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.1 // indirect
	github.com/rivo/uniseg v0.1.0 // indirect
	github.com/scritchley/orc v0.0.0-20210513144143-06dddf1ad665 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sijms/go-ora/v2 v2.4.20 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/snowflakedb/gosnowflake v1.6.9 // indirect
	github.com/xitongsys/parquet-go v1.6.2 // indirect
	github.com/xitongsys/parquet-go-source v0.0.0-20220315005136-aec0fe3e777c // indirect
	github.com/xuri/efp v0.0.0-20220407160117-ad0f7a785be8 // indirect
	github.com/xuri/excelize/v2 v2.6.0 // indirect
	github.com/xuri/nfp v0.0.0-20220409054826-5e722a1d9e22 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.opentelemetry.io/otel v1.7.0 // indirect
	go.opentelemetry.io/otel/trace v1.7.0 // indirect
	golang.org/x/crypto v0.0.0-20220518034528-6f7dac969898 // indirect
	golang.org/x/net v0.0.0-20220407224826-aac1ed45d8e3 // indirect
	golang.org/x/oauth2 v0.0.0-20220309155454-6242fa91716a // indirect
	golang.org/x/sys v0.0.0-20220429233432-b5fbb4746d32 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f // indirect
	gonum.org/v1/gonum v0.11.0 // indirect
	google.golang.org/api v0.74.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20220407144326-9054f6ed7bac // indirect
	google.golang.org/grpc v1.45.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
