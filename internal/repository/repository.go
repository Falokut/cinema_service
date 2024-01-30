package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("entity not found")
var ErrInvalidArgument = errors.New("invalid input data")

func NewPostgreDB(cfg DBConfig) (*sqlx.DB, error) {
	conStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBName, cfg.SSLMode)
	db, err := sqlx.Connect("pgx", conStr)

	if err != nil {
		return nil, err
	}

	return db, nil
}

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

type Hall struct {
	Id   int32  `db:"id" json:"id"`
	Type string `db:"hall_type" json:"hall_type"`
	Name string `db:"name" json:"name"`
	Size uint32 `db:"size" json:"size"`
}

type Cinema struct {
	ID          int32    `json:"id" db:"id"`
	Name        string   `json:"name" db:"name"`
	Address     string   `json:"address" db:"address"`
	Coordinates GeoPoint `json:"coordinates" db:"coordinates"`
}

type City struct {
	ID   int32  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type MoviesScreenings struct {
	MovieID         int32    `json:"movie_id" db:"movie_id"`
	ScreeningsTypes []string `json:"screenings_types" db:"screenings_types"`
	HallsTypes      []string `json:"halls_types" db:"halls_types"`
}

type Screening struct {
	ScreeningID   int64     `json:"id" db:"id"`
	MovieID       int32     `json:"movie_id" db:"movie_id"`
	ScreeningType string    `json:"screening_type" db:"screening_type"`
	HallID        int32     `json:"hall_id" db:"hall_id"`
	TicketPrice   string    `json:"ticket_price" db:"ticket_price"`
	StartTime     time.Time `json:"start_time" db:"start_time"`
}

type Place struct {
	Row      int32   `json:"row" db:"row"`
	Seat     int32   `json:"seat" db:"seat"`
	GridPosX float32 `json:"grid_pos_x" db:"grid_pos_x"`
	GridPosY float32 `json:"grid_pos_y" db:"grid_pos_y"`
}

type CinemaRepository interface {
	// Returns cinemas in the city.
	GetCinemasInCity(ctx context.Context, id int32) ([]Cinema, error)

	// Returns all cities where there are cinemas.
	GetCinemasCities(ctx context.Context) ([]City, error)

	// Returns all movies that are in the cinema screenings in a particular cinema.
	GetMoviesScreenings(ctx context.Context, cinemaID int32, startPeriod, endPeriod time.Time) ([]MoviesScreenings, error)

	// Returns all movies that are in the cinema screenings.
	GetAllMoviesScreenings(ctx context.Context, startPeriod, endPeriod time.Time) ([]MoviesScreenings, error)

	// Returns all movies that are in the cinema screenings in particular cities.
	GetMoviesScreeningsInCities(ctx context.Context, citiesIds []int32, startPeriod, endPeriod time.Time) ([]MoviesScreenings, error)

	//Returns all screenings for a movie in a specific cinema.
	GetScreenings(ctx context.Context, cinemaID, movieID int32, startPeriod, endPeriod time.Time) ([]Screening, error)

	// Returns the configuration of the hall.
	GetHallConfiguraion(ctx context.Context, id int32) ([]Place, error)

	// Returns info for the halls with specified ids (without configuration).
	GetHalls(ctx context.Context, ids []int32) ([]Hall, error)
}

type CinemaCache interface {
	// Returns cinemas in the city.
	GetCinemasInCity(ctx context.Context, id int32) ([]Cinema, error)

	// Returns all cities where there are cinemas.
	GetCinemasCities(ctx context.Context) ([]City, error)

	// Returns the configuration of the hall.
	GetHallConfiguraion(ctx context.Context, id int32) ([]Place, error)

	// Returns info for the halls with specified ids and not founded ids (without configuration).
	GetHalls(ctx context.Context, ids []int32) ([]Hall, []int32, error)

	CacheCinemasInCity(ctx context.Context, id int32, cinemas []Cinema, ttl time.Duration) error
	CacheCinemasCities(ctx context.Context, cities []City, ttl time.Duration) error
	CacheHallConfiguraion(ctx context.Context, id int32, places []Place, ttl time.Duration) error
	CacheHalls(ctx context.Context, halls []Hall, ttl time.Duration) error
}
