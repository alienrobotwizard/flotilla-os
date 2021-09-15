module github.com/alienrobotwizard/flotilla-os

go 1.12

require (
	github.com/DataDog/datadog-go v3.2.1-0.20191119163752-87e3273d8c23+incompatible
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/aws/aws-sdk-go v1.40.18
	github.com/gin-gonic/gin v1.7.4
	github.com/go-kit/kit v0.9.0
	github.com/go-redis/redis v6.15.6+incompatible
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.4-0.20190701202633-d83b6ffe499a
	github.com/jmoiron/sqlx v1.2.1-0.20190426154859-38398a30ed85
	github.com/lib/pq v1.10.2
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/pkg/errors v0.9.1
	github.com/rs/cors v1.6.1-0.20190613161432-33ffc0734c60
	github.com/spf13/viper v1.4.1-0.20190614151712-3349bd9cc288
	github.com/stitchfix/flotilla-os v0.0.0-20210902062707-a36f97aa656f
	github.com/stretchr/testify v1.7.0
	github.com/xeipuuv/gojsonschema v0.0.0-20180618132009-1d523034197f
	go.uber.org/multierr v1.5.0
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
	gorm.io/driver/postgres v1.1.1
	gorm.io/gorm v1.21.15
	k8s.io/api v0.0.0-20191121015604-11707872ac1c
	k8s.io/apimachinery v0.0.0-20191121015412-41065c7a8c2a
	k8s.io/client-go v0.0.0-20191121015835-571c0ef67034
	k8s.io/metrics v0.0.0-20191121021546-b1134fd1210c
)
