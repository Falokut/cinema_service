package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/Falokut/cinema_service/internal/config"
	"github.com/Falokut/cinema_service/internal/repository"
	"github.com/Falokut/cinema_service/internal/service"
	cinema_service "github.com/Falokut/cinema_service/pkg/cinema_service/v1/protos"
	jaegerTracer "github.com/Falokut/cinema_service/pkg/jaeger"
	"github.com/Falokut/cinema_service/pkg/metrics"
	server "github.com/Falokut/grpc_rest_server"
	"github.com/Falokut/healthcheck"
	logging "github.com/Falokut/online_cinema_ticket_office.loggerwrapper"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/opentracing/opentracing-go"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func main() {
	logging.NewEntry(logging.ConsoleOutput)
	logger := logging.GetLogger()
	cfg := config.GetConfig()

	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Logger.SetLevel(logLevel)

	tracer, closer, err := jaegerTracer.InitJaeger(cfg.JaegerConfig)
	if err != nil {
		logger.Errorf("Shutting down, error while creating tracer %v", err)
		return
	}
	logger.Info("Jaeger connected")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	logger.Info("Metrics initializing")
	metric, err := metrics.CreateMetrics(cfg.PrometheusConfig.Name)
	if err != nil {
		logger.Errorf("Shutting down, error while creating metrics %v", err)
		return
	}

	go func() {
		logger.Info("Metrics server running")
		if err := metrics.RunMetricServer(cfg.PrometheusConfig.ServerConfig); err != nil {
			logger.Errorf("Shutting down, error while running metrics server %v", err)
			return
		}
	}()

	cinemaDB, err := repository.NewPostgreDB(cfg.DBConfig)
	if err != nil {
		logger.Errorf("Shutting down, connection to the database not established %v", err)
		return
	}
	defer cinemaDB.Close()

	cinemaRdb, err := repository.NewRedisCache(&redis.Options{
		Network:  cfg.CinemasCache.Network,
		Addr:     cfg.CinemasCache.Addr,
		DB:       cfg.CinemasCache.DB,
		Password: cfg.CinemasCache.Password,
	})
	if err != nil {
		logger.Errorf("Shutting down, connection to the cinema cache not established %v", err)
		return
	}
	defer cinemaRdb.Close()

	citiesRdb, err := repository.NewRedisCache(&redis.Options{
		Network:  cfg.CitiesCache.Network,
		Addr:     cfg.CitiesCache.Addr,
		DB:       cfg.CitiesCache.DB,
		Password: cfg.CitiesCache.Password,
	})
	if err != nil {
		logger.Errorf("Shutting down, connection to the сities cache not established %v", err)
		return
	}
	defer citiesRdb.Close()

	hallsRdb, err := repository.NewRedisCache(&redis.Options{
		Network:  cfg.HallsCache.Network,
		Addr:     cfg.HallsCache.Addr,
		DB:       cfg.HallsCache.DB,
		Password: cfg.HallsCache.Password,
	})
	if err != nil {
		logger.Errorf("Shutting down, connection to the halls cache not established %v", err)
		return
	}
	defer hallsRdb.Close()

	cinemaCache := repository.NewCinemaCache(logger.Logger, cinemaRdb, citiesRdb, hallsRdb)
	go func() {
		logger.Info("Healthcheck initializing")
		healthcheckManager := healthcheck.NewHealthManager(logger.Logger,
			[]healthcheck.HealthcheckResource{cinemaDB, cinemaCache}, cfg.HealthcheckPort, nil)
		if err := healthcheckManager.RunHealthcheckEndpoint(); err != nil {
			logger.Error(err)
			return
		}
	}()

	cinemaRepo := repository.NewCinemaRepository(logger.Logger, cinemaDB)
	repository := service.NewCinemaRepositoryWrapper(logger.Logger, cinemaRepo, cinemaCache,
		service.CacheConfig{
			HallConfigurationTTL: cfg.HallsCache.TTL,
			CinemasTTL:           cfg.CinemasCache.TTL,
			CitiesTTL:            cfg.CitiesCache.TTL,
		}, metric)

	service := service.NewCinemaService(logger.Logger, repository)
	s := server.NewServer(logger.Logger, service)
	s.Run(getListenServerConfig(cfg), metric, nil, nil)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGTERM)

	<-quit
	s.Shutdown()
}

func getListenServerConfig(cfg *config.Config) server.Config {
	return server.Config{
		Mode:        cfg.Listen.Mode,
		Host:        cfg.Listen.Host,
		Port:        cfg.Listen.Port,
		ServiceDesc: &cinema_service.CinemaServiceV1_ServiceDesc,
		RegisterRestHandlerServer: func(ctx context.Context, mux *runtime.ServeMux, service any) error {
			serv, ok := service.(cinema_service.CinemaServiceV1Server)
			if !ok {
				return errors.New("can't convert")
			}

			return cinema_service.RegisterCinemaServiceV1HandlerServer(context.Background(),
				mux, serv)
		},
	}
}
