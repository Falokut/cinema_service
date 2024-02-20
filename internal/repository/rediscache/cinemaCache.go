package rediscache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Falokut/cinema_service/internal/models"
	"github.com/opentracing/opentracing-go"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
)

type CinemaCache struct {
	logger                 *logrus.Logger
	citiesCinemasRdb       *redis.Client
	citiesRdb              *redis.Client
	hallsConfigurationsRdb *redis.Client
	hallsRdb               *redis.Client
	cinemasRdb             *redis.Client
}

func NewCinemaCache(logger *logrus.Logger, cinemasCitiesRdb, cinemasRdb, citiesRdb, hallsConfigurationsRdb, hallsRdb *redis.Client) *CinemaCache {
	return &CinemaCache{
		logger:                 logger,
		citiesCinemasRdb:       cinemasCitiesRdb,
		cinemasRdb:             cinemasRdb,
		citiesRdb:              citiesRdb,
		hallsConfigurationsRdb: hallsConfigurationsRdb,
		hallsRdb:               hallsRdb,
	}
}

func (c *CinemaCache) PingContext(ctx context.Context) error {
	if err := c.cinemasRdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("error while pinging cinemas cache: %w", err)
	}
	if err := c.citiesCinemasRdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("error while pinging cities cinemas cache: %w", err)
	}
	if err := c.citiesRdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("error while pinging cities cache: %w", err)
	}
	if err := c.hallsRdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("error while pinging halls cache: %w", err)
	}
	if err := c.hallsConfigurationsRdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("error while pinging halls configuration cache: %w", err)
	}
	return nil
}

func (c *CinemaCache) GetCinema(ctx context.Context, id int32) (models.Cinema, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaCacheGetCinema")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	data, err := c.cinemasRdb.Get(ctx, fmt.Sprint(id)).Bytes()
	if err != nil {
		return models.Cinema{}, err
	}
	var cinema models.Cinema
	err = json.Unmarshal(data, &cinema)
	if err != nil {
		c.logger.Errorf("error while unmarchalling cinema %s", err.Error())
		return models.Cinema{}, err
	}

	return cinema, nil
}

func (c *CinemaCache) CacheCinema(ctx context.Context, cinema models.Cinema, ttl time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaCache.CacheCinema")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)
	data, err := json.Marshal(cinema)
	if err != nil {
		return err
	}

	err = c.cinemasRdb.Set(ctx, fmt.Sprint(cinema.ID), data, ttl).Err()
	return err
}

func (c *CinemaCache) GetCinemasInCity(ctx context.Context, id int32) ([]models.Cinema, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaCache.GetCinemasInCity")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	data, err := c.citiesCinemasRdb.Get(ctx, fmt.Sprint(id)).Bytes()
	if err != nil {
		c.logger.Errorf("error while getting cinemas in city %s", err.Error())
		return []models.Cinema{}, err
	}

	var cinemas []models.Cinema
	err = json.Unmarshal(data, &cinemas)
	if err != nil {
		c.logger.Errorf("error while unmarchalling cinemas in city %s", err.Error())
		return []models.Cinema{}, err
	}

	return cinemas, nil
}

func (c *CinemaCache) GetCinemasCities(ctx context.Context) ([]models.City, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaCache.CacheCinemasInCity")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	var keys []string
	c.citiesRdb.Keys(ctx, "*").ScanSlice(&keys)
	if err != nil {
		c.logger.Errorf("error while getting cinemas cities keys %s", err.Error())
		return []models.City{}, err
	}

	redisData, err := c.citiesRdb.MGet(ctx, keys...).Result()
	if err != nil {
		c.logger.Errorf("error while getting cinemas cities %s", err.Error())
		return []models.City{}, err
	}

	var cities = make([]models.City, 0, len(redisData))
	for _, data := range redisData {
		var city models.City
		err = json.Unmarshal([]byte(data.(string)), &city)
		if err != nil {
			c.logger.Errorf("error while unmarchalling cinemas in city %s", err.Error())
			return []models.City{}, err
		}

		cities = append(cities, city)
	}

	return cities, nil
}

func (c *CinemaCache) GetHallConfiguraion(ctx context.Context, id int32) ([]models.Place, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaCache.GetCinemasInCity")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	data, err := c.hallsConfigurationsRdb.Get(ctx, fmt.Sprint(id)).Bytes()
	if err != nil {
		c.logger.Errorf("error while getting places %s", err.Error())
		return []models.Place{}, err
	}

	var places []models.Place
	err = json.Unmarshal(data, &places)
	if err != nil {
		c.logger.Errorf("error while unmarchalling places %s", err.Error())
		return []models.Place{}, err
	}

	return places, nil
}

func (c *CinemaCache) CacheCinemasInCity(ctx context.Context, id int32, cinemas []models.Cinema, ttl time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaCache.CacheCinemasInCity")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	data, err := json.Marshal(cinemas)
	if err != nil {
		return err
	}

	err = c.citiesCinemasRdb.Set(ctx, fmt.Sprint(id), data, ttl).Err()
	return err
}

func (c *CinemaCache) CacheCinemasCities(ctx context.Context, cities []models.City, ttl time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaCache.CacheCinemasCities")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	tx := c.citiesRdb.Pipeline()
	for _, city := range cities {
		toCache, err := json.Marshal(city)
		if err != nil {
			return err
		}
		tx.Set(ctx, fmt.Sprint(city.ID), toCache, ttl)
	}
	_, err = tx.Exec(ctx)
	return err
}

func (c *CinemaCache) CacheHallConfiguraion(ctx context.Context, id int32, places []models.Place, ttl time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaCache.CacheHallConfiguraion")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	toCache, err := json.Marshal(places)
	if err != nil {
		return err
	}

	err = c.hallsConfigurationsRdb.Set(ctx, fmt.Sprint(id), toCache, ttl).Err()
	return err
}

func (c *CinemaCache) CacheHalls(ctx context.Context, halls []models.Hall, ttl time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaCache.CacheHalls")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	tx := c.hallsRdb.Pipeline()
	for _, hall := range halls {
		toCache, err := json.Marshal(hall)
		if err != nil {
			return err
		}
		tx.Set(ctx, fmt.Sprint(hall.Id), toCache, ttl)
	}
	_, err = tx.Exec(ctx)
	return err
}

func (c *CinemaCache) GetHalls(ctx context.Context, ids []int32) ([]models.Hall, []int32, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaCache.GetHalls")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	keys := make([]string, len(ids))
	hallsIds := make(map[int32]struct{}, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprint(id)
		hallsIds[id] = struct{}{}
	}

	halls, err := c.hallsRdb.MGet(ctx, keys...).Result()
	if err != nil {
		return []models.Hall{}, []int32{}, err
	}

	var cachedHalls = make([]models.Hall, 0, len(halls))
	for _, cached := range halls {
		if cached == nil {
			continue
		}

		hall := models.Hall{}
		err = json.Unmarshal([]byte(cached.(string)), &hall)
		if err != nil {
			return []models.Hall{}, []int32{}, err
		}
		delete(hallsIds, hall.Id)
		cachedHalls = append(cachedHalls, hall)
	}

	return cachedHalls, maps.Keys(hallsIds), nil
}
