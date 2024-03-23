package config

import (
	"sync"
	"time"

	"github.com/Falokut/cinema_service/internal/repository"
	"github.com/Falokut/cinema_service/pkg/jaeger"
	"github.com/Falokut/cinema_service/pkg/logging"
	"github.com/Falokut/cinema_service/pkg/metrics"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel        string `yaml:"log_level" env:"LOG_LEVEL"`
	HealthcheckPort string `yaml:"healthcheck_port" env:"HEALTHCHECK_PORT"`
	Listen          struct {
		Host string `yaml:"host" env:"HOST"`
		Port string `yaml:"port" env:"PORT"`
		Mode string `yaml:"server_mode" env:"SERVER_MODE"` // support GRPC, REST, BOTH
	} `yaml:"listen"`

	PrometheusConfig struct {
		Name         string                      `yaml:"service_name" ENV:"PROMETHEUS_SERVICE_NAME"`
		ServerConfig metrics.MetricsServerConfig `yaml:"server_config"`
	} `yaml:"prometheus"`

	DBConfig     repository.DBConfig `yaml:"db_config"`
	JaegerConfig jaeger.Config       `yaml:"jaeger"`

	CinemasCache struct {
		Network  string        `yaml:"network" env:"CINEMA_CACHE_NETWORK"`
		Addr     string        `yaml:"addr" env:"CINEMA_CACHE_ADDR"`
		DB       int           `yaml:"db" env:"CINEMA_CACHE_DB"`
		Password string        `yaml:"password" env:"CINEMA_CACHE_PASSWORD"`
		TTL      time.Duration `yaml:"ttl"`
	} `yaml:"cinemas_cache"`

	CitiesCinemasCache struct {
		Network  string        `yaml:"network" env:"CITIES_CINEMA_CACHE_NETWORK"`
		Addr     string        `yaml:"addr" env:"CITIES_CINEMA_CACHE_ADDR"`
		DB       int           `yaml:"db" env:"CITIES_CINEMA_CACHE_DB"`
		Password string        `yaml:"password" env:"CITIES_CINEMA_CACHE_PASSWORD"`
		TTL      time.Duration `yaml:"ttl"`
	} `yaml:"cities_cinemas_cache"`

	CitiesCache struct {
		Network  string        `yaml:"network" env:"CITIES_CACHE_NETWORK"`
		Addr     string        `yaml:"addr" env:"CITIES_CACHE_ADDR"`
		DB       int           `yaml:"db" env:"CITIES_CACHE_DB"`
		Password string        `yaml:"password" env:"CITIES_CACHE_PASSWORD"`
		TTL      time.Duration `yaml:"ttl"`
	} `yaml:"cities_cache"`
	HallsCache struct {
		Network  string        `yaml:"network" env:"HALLS_CACHE_NETWORK"`
		Addr     string        `yaml:"addr" env:"HALLS_CACHE_ADDR"`
		DB       int           `yaml:"db" env:"HALLS_CACHE_DB"`
		Password string        `yaml:"password" env:"HALLS_CACHE_PASSWORD"`
		TTL      time.Duration `yaml:"ttl"`
	} `yaml:"halls_cache"`

	HallsConfigurationCache struct {
		Network  string        `yaml:"network" env:"HALLS_CONFIGURATIONS_CACHE_NETWORK"`
		Addr     string        `yaml:"addr" env:"HALLS_CONFIGURATIONS_CACHE_ADDR"`
		DB       int           `yaml:"db" env:"HALLS_CONFIGURATIONS_CACHE_DB"`
		Password string        `yaml:"password" env:"HALLS_CONFIGURATIONS_CACHE_PASSWORD"`
		TTL      time.Duration `yaml:"ttl"`
	} `yaml:"halls_configurations_cache"`
}

var instance *Config
var once sync.Once

const configsPath = "configs/"

func GetConfig() *Config {
	once.Do(func() {
		logger := logging.GetLogger()
		instance = &Config{}

		if err := cleanenv.ReadConfig(configsPath+"config.yml", instance); err != nil {
			help, _ := cleanenv.GetDescription(instance, nil)
			logger.Fatal(help, " ", err)
		}
	})

	return instance
}
