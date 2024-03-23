package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/Falokut/cinema_service/internal/config"
	"github.com/Falokut/cinema_service/internal/handler"
	"github.com/Falokut/cinema_service/internal/repository"
	"github.com/Falokut/cinema_service/internal/repository/postgresrepository"
	"github.com/Falokut/cinema_service/internal/repository/rediscache"
	"github.com/Falokut/cinema_service/internal/service"
	cinema_service "github.com/Falokut/cinema_service/pkg/cinema_service/v1/protos"
	jaegerTracer "github.com/Falokut/cinema_service/pkg/jaeger"
	"github.com/Falokut/cinema_service/pkg/logging"
	"github.com/Falokut/cinema_service/pkg/metrics"
	server "github.com/Falokut/grpc_rest_server"
	"github.com/Falokut/healthcheck"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/opentracing/opentracing-go"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

func initHealthcheck(cfg *config.Config, shutdown chan error, resources []healthcheck.HealthcheckResource) {
	logger := logging.GetLogger()
	logger.Info("Healthcheck initializing")
	healthcheckManager := healthcheck.NewHealthManager(logger.Logger,
		resources, cfg.HealthcheckPort, nil)
	go func() {
		logger.Info("Healthcheck server running")
		if err := healthcheckManager.RunHealthcheckEndpoint(); err != nil {
			logger.Errorf("Shutting down, can't run healthcheck endpoint %v", err)
			shutdown <- err
		}
	}()
}

func initMetrics(cfg *config.Config, shutdown chan error) (metrics.Metrics, error) {
	logger := logging.GetLogger()

	tracer, closer, err := jaegerTracer.InitJaeger(cfg.JaegerConfig)
	if err != nil {
		logger.Errorf("Shutting down, error while creating tracer %v", err)
		return nil, err
	}

	logger.Info("Jaeger connected")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	logger.Info("Metrics initializing")
	metric, err := metrics.CreateMetrics(cfg.PrometheusConfig.Name)
	if err != nil {
		logger.Errorf("Shutting down, error while creating metrics %v", err)
		return nil, err
	}

	go func() {
		logger.Info("Metrics server running")
		if err := metrics.RunMetricServer(cfg.PrometheusConfig.ServerConfig); err != nil {
			logger.Errorf("Shutting down, error while running metrics server %v", err)
			shutdown <- err
		}
	}()

	return metric, nil
}

func main() {
	logging.NewEntry(logging.ConsoleOutput)
	logger := logging.GetLogger()
	cfg := config.GetConfig()

	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Logger.SetLevel(logLevel)

	shutdown := make(chan error, 1)
	metric, err := initMetrics(cfg, shutdown)
	if err != nil {
		return
	}

	cinemaDB, err := postgresrepository.NewPostgreDB(&cfg.DBConfig)
	if err != nil {
		logger.Errorf("Shutting down, connection to the database not established %v", err)
		return
	}
	defer cinemaDB.Close()

	citiesCinemas, err := rediscache.NewRedisCache(&redis.Options{
		Network:  cfg.CitiesCinemasCache.Network,
		Addr:     cfg.CitiesCinemasCache.Addr,
		DB:       cfg.CitiesCinemasCache.DB,
		Password: cfg.CitiesCinemasCache.Password,
	})
	if err != nil {
		logger.Errorf("Shutting down, connection to the cities cinemas cache not established %v", err)
		return
	}
	defer citiesCinemas.Close()

	cinemasRdb, err := rediscache.NewRedisCache(&redis.Options{
		Network:  cfg.CinemasCache.Network,
		Addr:     cfg.CinemasCache.Addr,
		DB:       cfg.CinemasCache.DB,
		Password: cfg.CinemasCache.Password,
	})
	if err != nil {
		logger.Errorf("Shutting down, connection to the cinema cache not established %v", err)
		return
	}
	defer cinemasRdb.Close()

	citiesRdb, err := rediscache.NewRedisCache(&redis.Options{
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

	hallsRdb, err := rediscache.NewRedisCache(&redis.Options{
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

	hallsConfigurationsRdb, err := rediscache.NewRedisCache(&redis.Options{
		Network:  cfg.HallsConfigurationCache.Network,
		Addr:     cfg.HallsConfigurationCache.Addr,
		DB:       cfg.HallsConfigurationCache.DB,
		Password: cfg.HallsConfigurationCache.Password,
	})
	if err != nil {
		logger.Errorf("Shutting down, connection to the halls configurations cache not established %v", err)
		return
	}
	defer hallsConfigurationsRdb.Close()

	cinemaCache := rediscache.NewCinemaCache(logger.Logger, citiesCinemas, cinemasRdb, citiesRdb,
		hallsConfigurationsRdb, hallsRdb, metric)

	initHealthcheck(cfg, shutdown, []healthcheck.HealthcheckResource{cinemaDB, cinemaCache})

	repo := postgresrepository.NewCinemaRepository(logger.Logger, cinemaDB)
	repositoryWithCache := repository.NewcinemaRepositoryWithCache(logger.Logger, repo, cinemaCache,
		repository.CacheConfig{
			HallConfigurationTTL: cfg.HallsCache.TTL,
			CinemasTTL:           cfg.CinemasCache.TTL,
			CitiesTTL:            cfg.CitiesCache.TTL,
			HallsTTL:             cfg.HallsCache.TTL,
			CitiesCinemasTTL:     cfg.CitiesCinemasCache.TTL,
		})

	s := service.NewCinemaService(repositoryWithCache)
	h := handler.NewCinemaServiceHandler(s)
	logger.Info("Server initializing")
	serv := server.NewServer(logger.Logger, h)
	go func() {
		if err := serv.Run(getListenServerConfig(cfg), metric, nil, nil); err != nil {
			logger.Errorf("Shutting down, error while running server %s", err.Error())
			shutdown <- err
			return
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGTERM)

	select {
	case <-quit:
		break
	case <-shutdown:
		break
	}

	serv.Shutdown()
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

			return cinema_service.RegisterCinemaServiceV1HandlerServer(ctx,
				mux, serv)
		},
	}
}
