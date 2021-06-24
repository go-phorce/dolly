module github.com/go-phorce/dolly

go 1.16

require (
	github.com/DataDog/datadog-go v4.8.0+incompatible
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/aws/aws-sdk-go v1.38.66
	github.com/cloudflare/cfssl v1.6.0
	github.com/go-phorce/cov-report v1.1.1-0.20200622030546-3fb510c4b1ba
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0
	github.com/jinzhu/copier v0.3.2
	github.com/jteeuwen/go-bindata v3.0.7+incompatible
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/testing v0.0.0-20210324180055-18c50b0c2098 // indirect
	github.com/julienschmidt/httprouter v1.3.0
	github.com/lib/pq v1.7.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.0 // indirect
	github.com/mattn/goveralls v0.0.9
	github.com/miekg/pkcs11 v1.0.3
	github.com/prometheus/client_golang v1.11.0
	github.com/rs/cors v1.7.0
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/ugorji/go/codec v1.2.6
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/text v0.3.6 // indirect
	golang.org/x/tools v0.1.4
	google.golang.org/grpc v1.38.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)

replace golang.org/x/text => golang.org/x/text v0.3.6
