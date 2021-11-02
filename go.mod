module github.com/tidepool-org/tide-whisperer

go 1.15

// Commit id relative to go-common@dblp.0.9.0
replace github.com/tidepool-org/go-common => github.com/mdblp/go-common v0.7.2-0.20210611071916-2fe8363c8b02

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/google/uuid v1.1.2
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mdblp/hydrophone v0.8.0
	github.com/mdblp/tide-whisperer-v2 v0.0.0-20211029144901-152199a587e3
	github.com/prometheus/client_golang v1.11.0
	github.com/swaggo/swag v1.7.4
	github.com/tidepool-org/go-common v0.0.0
	gitlab.com/msvechla/mux-prometheus v0.0.2
	go.mongodb.org/mongo-driver v1.7.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)
