package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Falokut/cinema_service/internal/models"
	"github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("entity not found")
var ErrInvalidArgument = errors.New("invalid input data")

func NewRedisCache(opt *redis.Options) (*redis.Client, error) {
	rdb := redis.NewClient(opt)
	if rdb == nil {
		return nil, errors.New("can't create new redis client")
	}

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("connection is not established: %s", err.Error())
	}

	return rdb, nil
}

type DBConfig struct {
	Host     string `yaml:"host" env:"DB_HOST"`
	Port     string `yaml:"port" env:"DB_PORT"`
	Username string `yaml:"username" env:"DB_USERNAME"`
	Password string `yaml:"password" env:"DB_PASSWORD"`
	DBName   string `yaml:"db_name" env:"DB_NAME"`
	SSLMode  string `yaml:"ssl_mode" env:"DB_SSL_MODE"`
}

type CinemaRepository interface {
	GetScreening(ctx context.Context, id int64) (models.Screening, error)
	// Returns cinemas in the city.
	GetCinemasInCity(ctx context.Context, id int32) ([]models.Cinema, error)

	// Returns all cities where there are cinemas.
	GetCinemasCities(ctx context.Context) ([]models.City, error)

	// Returns all movies that are in the cinema screenings in a particular cinema.
	GetMoviesScreenings(ctx context.Context, cinemaID int32, startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error)

	// Returns all screenings for a movie in a specific city.
	GetCityScreenings(ctx context.Context, cityId, movieId int32, startPeriod, endPeriod time.Time) ([]models.CityScreening, error)

	// Returns all movies that are in the cinema screenings.
	GetAllMoviesScreenings(ctx context.Context, startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error)

	// Returns all movies that are in the cinema screenings in particular cities.
	GetMoviesScreeningsInCities(ctx context.Context, citiesIds []int32, startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error)

	//Returns all screenings for a movie in a specific cinema.
	GetScreenings(ctx context.Context, cinemaID, movieID int32, startPeriod, endPeriod time.Time) ([]models.Screening, error)

	// Returns the configuration of the hall.
	GetHallConfiguraion(ctx context.Context, id int32) ([]models.Place, error)

	// Returns info for the halls with specified ids (without configuration).
	GetHalls(ctx context.Context, ids []int32) ([]models.Hall, error)

	// Returns cinema with specified id.
	GetCinema(ctx context.Context, id int32) (models.Cinema, error)
}

type CinemaCache interface {
	// Returns cinemas in the city.
	GetCinemasInCity(ctx context.Context, id int32) ([]models.Cinema, error)

	// Returns all cities where there are cinemas.
	GetCinemasCities(ctx context.Context) ([]models.City, error)

	// Returns the configuration of the hall.
	GetHallConfiguraion(ctx context.Context, id int32) ([]models.Place, error)

	// Returns info for the halls with specified ids and not founded ids (without configuration).
	GetHalls(ctx context.Context, ids []int32) ([]models.Hall, []int32, error)

	// Returns cinema with specified id.
	GetCinema(ctx context.Context, id int32) (models.Cinema, error)

	CacheCinemasInCity(ctx context.Context, id int32, cinemas []models.Cinema, ttl time.Duration) error
	CacheCinemasCities(ctx context.Context, cities []models.City, ttl time.Duration) error
	CacheHallConfiguraion(ctx context.Context, id int32, places []models.Place, ttl time.Duration) error
	CacheHalls(ctx context.Context, halls []models.Hall, ttl time.Duration) error
	CacheCinema(ctx context.Context, cinema models.Cinema, ttl time.Duration) error
}
