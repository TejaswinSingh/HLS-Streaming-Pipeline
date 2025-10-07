package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	gps "github.com/Resolyte-official/service-gps/pkg"
	grpcClient "github.com/Resolyte-official/service-gps/pkg/client/grpc"
	"github.com/Resolyte-official/service-gps/pkg/config"
	mongodb "github.com/Resolyte-official/service-gps/pkg/db/mongo"
	"github.com/Resolyte-official/service-gps/pkg/endpoints"
	"github.com/Resolyte-official/service-gps/pkg/logging"
	"github.com/Resolyte-official/service-gps/pkg/metrics"
	storage "github.com/Resolyte-official/service-gps/pkg/storage/s3"
	"github.com/Resolyte-official/service-gps/pkg/transport"
	"github.com/Resolyte-official/service-gps/pkg/trip"
	"github.com/Resolyte-official/service-gps/pkg/utils"
	"github.com/oklog/oklog/pkg/group"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	envConfig := config.NewEnvironmentConfiguration()

	var lw logging.LogWriter
	if envConfig.Env.IsProduction() {
		lw = logging.NewJsonLogWriter(os.Stdout, logging.INFO)
	} else {
		lw = logging.NewTextLogWriter(os.Stdout, logging.INFO)
	}

	mainLogger := logging.NewContextLogger(lw, "Main")

	mainLogger.Infof("starting gps in %s mode", envConfig.Env)

	if envConfig.HttpMode.IsHttps() { // need certs
		requireFile("configs/cert.pem")
		requireFile("configs/key.pem")
	}

	creds := insecure.NewCredentials() // TODO: use mTLS
	govClient, err := grpcClient.NewGovernanceClient(envConfig.GovernanceGrpcAddr, creds)
	if err != nil {
		panic(err)
	}
	defer govClient.Close()

	mongoURI := mongodb.CreateMongoURI(&envConfig.Mongo)
	mongoService, err := mongodb.NewMongoService(mongoURI)
	if err != nil {
		panic(err)
	}

	s3svc, err := storage.NewS3Service(context.Background())
	if err != nil {
		panic(err)
	}

	var (
		corsConfig = &transport.CorsOptions{
			Origins:          envConfig.CorsOrigins,                                                                   // Allow only specified origin
			Methods:          []string{"GET", "POST", "OPTIONS", "PUT", "UPDATE", "DELETE"},                           // Allow common HTTP methods
			Headers:          []string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "X-Request-ID"}, // Allow common HTTP headers
			ExposeHeaders:    []string{"Content-Type", "X-Request-ID"},                                                // Allow any headers to be exposed
			AllowCredentials: true,                                                                                    // Allow credentials to be included
		}
		dbTripStorage = trip.NewDBTripStorage(mongoService, s3svc, &envConfig)
		fetchList     = utils.NewFetchList(lw, dbTripStorage)
		metrics       = metrics.NewGPSMetrics()
		service       = gps.NewService(&envConfig, mongoService, dbTripStorage, fetchList)
		eps           = endpoints.NewEndpointSet(service)
		httpHandler   = transport.NewHTTPHandler(eps, &envConfig, corsConfig, govClient, metrics, lw)
	)

	var g group.Group

	{
		// http
		httpAddr := net.JoinHostPort("0.0.0.0", envConfig.HttpPort) // net.JoinHostPort("localhost", envConfig.HttpPort)
		srv := &http.Server{
			Addr:    httpAddr,
			Handler: httpHandler,
		}
		g.Add(func() error {
			mainLogger.Infof("starting %s server on %s", envConfig.HttpMode, httpAddr)
			if envConfig.HttpMode.IsHttps() {
				return srv.ListenAndServeTLS("configs/cert.pem", "configs/key.pem")
			}
			return srv.ListenAndServe()
		}, func(err error) {
			mainLogger.ErrorNtf(err, "shutting down %s server", envConfig.HttpMode)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		})
	}

	{
		// fetch coordinates from gps devices and append them to the trips,
		// which are stored in mongodb
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			var wg sync.WaitGroup

			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					wg.Add(1)
					go func() {
						defer wg.Done()
						fetchList.FetchAndAppendCoordinates()
					}()
				case <-ctx.Done():
					wg.Wait() // waits for all go routines called above to finish
					return nil
				}
			}
		}, func(error) {
			cancel()
			mainLogger.ErrorNt(err, "fetch coordinates | stopping")
		})
	}

	{
		// mongo
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					err := mongoService.Ping(ctx)
					if err != nil {
						mainLogger.Error(err, "MongoService | status disconnected")
						// return err
					} else {
						mainLogger.Info("MongoService | status healthy")
					}
				case <-ctx.Done():
					return nil
				}
			}
		}, func(err error) {
			cancel()
			mainLogger.ErrorNt(err, "MongoService | disconnecting mongo  client")
			ctx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelTimeout()
			if err := mongoService.Disconnect(ctx); err != nil {
				mainLogger.ErrorNt(err, "MongoService | error disconnecting mongo client")
			}
		})
	}

	{
		// handles Ctrl+C
		cancelInterrupt := make(chan struct{})
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-c:
				mainLogger.Infof("received signal: %s", sig)
				return nil
			case <-cancelInterrupt:
				return nil
			}
		}, func(error) {
			close(cancelInterrupt)
		})
	}

	mainLogger.Fatal(g.Run(), "stopping gps")
}

func requireFile(path string) {
	_, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
}
