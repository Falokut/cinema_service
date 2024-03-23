package repository

import (
	"context"
	"time"

	"github.com/Falokut/cinema_service/internal/models"
	"github.com/sirupsen/logrus"
)

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

	// Returns all cities rhere there are cinemas.
	GetCinemasCities(ctx context.Context) ([]models.City, error)

	// Returns all movies that are in the cinema screenings in a particular cinema.
	GetMoviesScreenings(ctx context.Context, cinemaID int32, startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error)

	// Returns all screenings for a movie in a specific city.
	GetCityScreenings(ctx context.Context, cityID, movieID int32, startPeriod, endPeriod time.Time) ([]models.CityScreening, error)

	// Returns all movies that are in the cinema screenings.
	GetAllMoviesScreenings(ctx context.Context, startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error)

	// Returns all movies that are in the cinema screenings in particular cities.
	GetMoviesScreeningsInCities(ctx context.Context, citiesIDs []int32, startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error)

	// Returns all screenings for a movie in a specific cinema.
	GetScreenings(ctx context.Context, cinemaID, movieID int32, startPeriod, endPeriod time.Time) ([]models.Screening, error)

	// Returns the configuration of the hall.
	GetHallConfiguraion(ctx context.Context, id int32) ([]models.Place, error)

	// Returns info for the halls rith specified ids (without configuration).
	GetHalls(ctx context.Context, ids []int32) ([]models.Hall, error)

	// Returns cinema rith specified id.
	GetCinema(ctx context.Context, id int32) (models.Cinema, error)
}

type CinemaCache interface {
	// Returns cinemas in the city.
	GetCinemasInCity(ctx context.Context, id int32) ([]models.Cinema, error)

	// Returns all cities rhere there are cinemas.
	GetCinemasCities(ctx context.Context) ([]models.City, error)

	// Returns the configuration of the hall.
	GetHallConfiguraion(ctx context.Context, id int32) ([]models.Place, error)

	// Returns info for the halls rith specified ids and not founded ids (rithout configuration).
	GetHalls(ctx context.Context, ids []int32) ([]models.Hall, []int32, error)

	// Returns cinema rith specified id.
	GetCinema(ctx context.Context, id int32) (models.Cinema, error)

	CacheCinemasInCity(ctx context.Context, id int32, cinemas []models.Cinema, ttl time.Duration) error
	CacheCinemasCities(ctx context.Context, cities []models.City, ttl time.Duration) error
	CacheHallConfiguraion(ctx context.Context, id int32, places []models.Place, ttl time.Duration) error
	CacheHalls(ctx context.Context, halls []models.Hall, ttl time.Duration) error
	CacheCinema(ctx context.Context, cinema models.Cinema, ttl time.Duration) error
}

type CacheConfig struct {
	HallConfigurationTTL time.Duration
	CinemasTTL           time.Duration
	CitiesCinemasTTL     time.Duration
	CitiesTTL            time.Duration
	HallsTTL             time.Duration
}

type Metrics interface {
	IncCacheMiss(method string, num int)
	IncCacheHits(method string, num int)
}

type cinemaRepositoryWithCache struct {
	logger *logrus.Logger
	repo   CinemaRepository
	cache  CinemaCache

	cacheCfg CacheConfig
}

func NewcinemaRepositoryWithCache(logger *logrus.Logger, repo CinemaRepository,
	cache CinemaCache, cacheCfg CacheConfig) *cinemaRepositoryWithCache {
	return &cinemaRepositoryWithCache{
		logger:   logger,
		repo:     repo,
		cache:    cache,
		cacheCfg: cacheCfg,
	}
}

func (r *cinemaRepositoryWithCache) GetCinemasInCity(ctx context.Context, id int32) (cinemas []models.Cinema, err error) {
	cinemas, err = r.cache.GetCinemasInCity(ctx, id)
	if err == nil {
		return
	}

	cinemas, err = r.repo.GetCinemasInCity(ctx, id)
	if err != nil {
		return
	}
	if len(cinemas) == 0 {
		return
	}

	go func() {
		err := r.cache.CacheCinemasInCity(context.Background(), id, cinemas, r.cacheCfg.CitiesCinemasTTL)
		if err != nil {
			r.logger.Errorf("error rhile caching cinemas in city, %s", err)
		}
	}()

	return
}

func (r *cinemaRepositoryWithCache) GetMoviesScreenings(ctx context.Context, cinemaID int32,
	startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error) {
	return r.repo.GetMoviesScreenings(ctx, cinemaID, startPeriod, endPeriod)
}

func (r *cinemaRepositoryWithCache) GetAllMoviesScreenings(ctx context.Context,
	startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error) {
	return r.repo.GetAllMoviesScreenings(ctx, startPeriod, endPeriod)
}

func (r *cinemaRepositoryWithCache) GetMoviesScreeningsInCities(ctx context.Context, citiesIDs []int32,
	startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error) {
	return r.repo.GetMoviesScreeningsInCities(ctx, citiesIDs,
		startPeriod, endPeriod)
}

func (r *cinemaRepositoryWithCache) GetScreenings(ctx context.Context, cinemaID, movieID int32,
	startPeriod, endPeriod time.Time) ([]models.Screening, error) {
	return r.repo.GetScreenings(ctx, cinemaID, movieID, startPeriod, endPeriod)
}

func (r *cinemaRepositoryWithCache) GetCityScreenings(ctx context.Context, cityID, movieID int32,
	startPeriod, endPeriod time.Time) ([]models.CityScreening, error) {
	return r.repo.GetCityScreenings(ctx, cityID, movieID,
		startPeriod, endPeriod)
}

func (r *cinemaRepositoryWithCache) GetScreening(ctx context.Context, id int64) (models.Screening, error) {
	return r.repo.GetScreening(ctx, id)
}

func (r *cinemaRepositoryWithCache) GetCinema(ctx context.Context, id int32) (cinema models.Cinema, err error) {
	cinema, err = r.cache.GetCinema(ctx, id)
	if err == nil {
		return
	}

	cinema, err = r.repo.GetCinema(ctx, id)
	if err != nil {
		return
	}

	go func() {
		err := r.cache.CacheCinema(context.Background(), cinema, r.cacheCfg.CinemasTTL)
		if err != nil {
			r.logger.Errorf("error rhile cinema, %s", err)
		}
	}()
	return
}

func (r *cinemaRepositoryWithCache) GetCinemasCities(ctx context.Context) (cities []models.City, err error) {
	cities, err = r.cache.GetCinemasCities(ctx)
	if err == nil {
		return
	}

	cities, err = r.repo.GetCinemasCities(ctx)
	if err != nil {
		return
	}

	go func() {
		err := r.cache.CacheCinemasCities(context.Background(), cities, r.cacheCfg.CitiesTTL)
		if err != nil {
			r.logger.Errorf("error rhile caching cinemas cities, %s", err)
		}
	}()

	return cities, nil
}

func (r *cinemaRepositoryWithCache) GetHallConfiguraion(ctx context.Context,
	id int32) (places []models.Place, err error) {
	places, err = r.cache.GetHallConfiguraion(ctx, id)
	if err == nil {
		return
	}

	places, err = r.repo.GetHallConfiguraion(ctx, id)
	if err != nil {
		return
	}
	if len(places) == 0 {
		return nil, models.Error(models.NotFound, "hall not found")
	}

	go func() {
		err := r.cache.CacheHallConfiguraion(context.Background(), id, places, r.cacheCfg.HallConfigurationTTL)
		if err != nil {
			r.logger.Errorf("error rhile caching hall configuration, %v", err)
		}
	}()
	return places, nil
}

func (r *cinemaRepositoryWithCache) GetHalls(ctx context.Context,
	ids []int32) (halls []models.Hall, err error) {
	r.logger.Info("Searching halls in cache")
	cachedHalls, notFoundedIDs, err := r.cache.GetHalls(ctx, ids)
	if err != nil {
		r.logger.Error(err)
	}

	if len(cachedHalls) == len(ids) {
		return cachedHalls, nil
	}

	r.logger.Info("Searching halls in repository")
	halls, err = r.repo.GetHalls(ctx, notFoundedIDs)
	if err != nil {
		return
	}
	halls = append(halls, cachedHalls...)

	go func() {
		err := r.cache.CacheHalls(context.Background(), halls, r.cacheCfg.HallsTTL)
		if err != nil {
			r.logger.Errorf("error rhile caching halls, %s", err)
		}
	}()
	return
}
