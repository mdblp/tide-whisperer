// @title Tide-Whisperer API
// @version 0.7.4
// @description Data access API for Diabeloop's diabetes data as used by Blip
// @license.name BSD 2-Clause "Simplified" License
// @host api.android-qa.your-loops.dev
// @BasePath /data
// @accept json
// @produce json
// @schemes https
// @contact.name Diabeloop
// #contact.url https://www.diabeloop.com
// @contact.email platforms@diabeloop.fr

// @securityDefinitions.apikey Auth0
// @in header
// @name Authorization
package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tidepool-org/tide-whisperer/api"
	common2 "github.com/tidepool-org/tide-whisperer/common"
	"github.com/tidepool-org/tide-whisperer/infrastructure"
	"github.com/tidepool-org/tide-whisperer/usecase"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/mdblp/go-common/clients/auth"
	tideV2Client "github.com/mdblp/tide-whisperer-v2/v2/client/tidewhisperer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tidepool-org/go-common"
	"github.com/tidepool-org/go-common/clients"
	"github.com/tidepool-org/go-common/clients/disc"
	"github.com/tidepool-org/go-common/clients/mongo"
	"github.com/tidepool-org/go-common/clients/opa"
	muxprom "gitlab.com/msvechla/mux-prometheus/pkg/middleware"
)

type (
	// TWConfig holds the configuration for the `tide-whisperer` service
	TWConfig struct {
		clients.Config
		Service               disc.ServiceListing `json:"service"`
		Mongo                 mongo.Config        `json:"mongo"`
		common2.SchemaVersion `json:"schemaVersion"`
	}
)

func main() {
	var twconfig TWConfig
	logger := log.New(os.Stdout, api.DataAPIPrefix, log.LstdFlags|log.Lshortfile)

	if err := common.LoadEnvironmentConfig(
		[]string{"TIDEPOOL_TIDE_WHISPERER_SERVICE", "TIDEPOOL_TIDE_WHISPERER_ENV"},
		&twconfig,
	); err != nil {
		logger.Fatal("Problem loading config: ", err)
	}
	authSecret := os.Getenv("API_SECRET")
	if authSecret == "" {
		logger.Fatal("Env var API_SECRET is not provided or empty")
	}

	// AWS part configuration
	bucketSuffix := os.Getenv("BUCKET_SUFFIX")
	if bucketSuffix == "" {
		logger.Fatal("Env var BUCKET_SUFFIX is not provided or empty")
	}
	region := os.Getenv("REGION")
	if region == "" {
		region = "eu-west-1"
		logger.Println("Using default aws region: ", region)
	}

	url := os.Getenv("S3_ENDPOINT_URL")
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if url != "" {
			logger.Println("Using custom s3 endpoint: ", url)
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               url,
				SigningRegion:     region,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	awsconfig, err := config.LoadDefaultConfig(context.TODO(), config.WithEndpointResolverWithOptions(customResolver), config.WithRegion(region))
	if err != nil {
		logger.Fatal(err)
	}
	s3Client := s3.NewFromConfig(awsconfig)
	uploader, err := usecase.NewUploader(s3Client, bucketSuffix)

	authClient, err := auth.NewClient(authSecret)
	twconfig.Mongo.FromEnv()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	if err != nil {
		logger.Fatal(err)
	}

	permsClient := opa.NewClientFromEnv(httpClient)

	tideV2Client := tideV2Client.NewTideWhispererClientFromEnv(httpClient)

	/*
	 * Instrumentation setup
	 */
	instrumentation := muxprom.NewCustomInstrumentation(true, "dblp", "tidewhisperer", prometheus.DefBuckets, nil, prometheus.DefaultRegisterer)

	patientDataMongoRepository, err := infrastructure.NewPatientDataMongoRepository(&twconfig.Mongo, logger)
	if err != nil {
		logger.Fatal(err)
	}
	defer patientDataMongoRepository.Close()
	patientDataMongoRepository.Start()
	rtr := mux.NewRouter()

	rtr.Use(instrumentation.Middleware)
	rtr.Path("/metrics").Handler(promhttp.Handler())

	/*
	 * Data-Api setup
	 */

	envReadBasalBucket, err := strconv.ParseBool(os.Getenv("READ_BASAL_BUCKET"))
	if err == nil && envReadBasalBucket {
		logger.Print("environment variable READ_BASAL_BUCKET exported,started with set true")
	} else {
		logger.Print("environment variable READ_BASAL_BUCKET not exported, started with false")
	}

	dataUseCase := usecase.NewPatientDataUseCase(logger, tideV2Client, patientDataMongoRepository, envReadBasalBucket)
	exportUseCase := usecase.NewExporter(logger, dataUseCase, uploader)
	exportController := api.NewExportController(logger, exportUseCase)

	api := api.InitAPI(exportController, dataUseCase, patientDataMongoRepository, authClient, permsClient, twconfig.SchemaVersion, logger, tideV2Client)
	api.SetHandlers("", rtr)

	// ability to return compressed (gzip/deflate) responses if client browser accepts it
	// this is interesting to minimise network traffic especially if we expect to have long
	// responses such as what the GetData() route here can return
	gzipHandler := handlers.CompressHandler(rtr)

	done := make(chan bool)
	server := common.NewServer(&http.Server{
		Addr:    twconfig.Service.GetPort(),
		Handler: gzipHandler,
	})

	var start func() error
	if twconfig.Service.Scheme == "https" {
		sslSpec := twconfig.Service.GetSSLSpec()
		start = func() error { return server.ListenAndServeTLS(sslSpec.CertFile, sslSpec.KeyFile) }
	} else {
		start = func() error { return server.ListenAndServe() }
	}
	if err := start(); err != nil {
		logger.Fatal(err)
	}

	// Wait for SIGINT (Ctrl+C) or SIGTERM to stop the service
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			<-sigc
			patientDataMongoRepository.Close()
			server.Close()
			done <- true
		}
	}()

	<-done
}
