package rediscache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Falokut/cinema_service/internal/models"
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
	metrics                Metrics
}

func NewCinemaCache(logger *logrus.Logger,
	cinemasCitiesRdb,
	cinemasRdb,
	citiesRdb,
	hallsConfigurationsRdb,
	hallsRdb *redis.Client,
	metrics Metrics) *CinemaCache {
	return &CinemaCache{
		logger:                 logger,
		citiesCinemasRdb:       cinemasCitiesRdb,
		cinemasRdb:             cinemasRdb,
		citiesRdb:              citiesRdb,
		hallsConfigurationsRdb: hallsConfigurationsRdb,
		hallsRdb:               hallsRdb,
		metrics:                metrics,
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

func (c *CinemaCache) GetCinema(ctx context.Context, id int32) (cinema models.Cinema, err error) {
	defer c.updateMetrics(&err, "GetCinema")
	defer handleError(ctx, &err)
	defer c.logError(&err, "GetCinema")
	data, err := c.cinemasRdb.Get(ctx, fmt.Sprint(id)).Bytes()
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &cinema)
	if err != nil {
		return
	}

	return cinema, nil
}

func (c *CinemaCache) CacheCinema(ctx context.Context, cinema models.Cinema, ttl time.Duration) (err error) {
	defer handleError(ctx, &err)
	defer c.logError(&err, "CacheCinema")
	data, err := json.Marshal(cinema)
	if err != nil {
		return err
	}

	err = c.cinemasRdb.Set(ctx, fmt.Sprint(cinema.ID), data, ttl).Err()
	return err
}

func (c *CinemaCache) GetCinemasInCity(ctx context.Context, cityID int32) (cinemas []models.Cinema, err error) {
	defer c.updateMetrics(&err, "GetCinemasInCity")
	defer handleError(ctx, &err)
	defer c.logError(&err, "GetCinemasInCity")
	data, err := c.citiesCinemasRdb.Get(ctx, fmt.Sprint(cityID)).Bytes()
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &cinemas)
	if err != nil {
		return
	}

	return cinemas, nil
}

func (c *CinemaCache) GetCinemasCities(ctx context.Context) (cities []models.City, err error) {
	defer c.updateMetrics(&err, "GetCinemasCities")
	defer handleError(ctx, &err)
	defer c.logError(&err, "GetCinemasCities")
	var keys []string
	err = c.citiesRdb.Keys(ctx, "*").ScanSlice(&keys)
	if err != nil {
		return
	}

	redisData, err := c.citiesRdb.MGet(ctx, keys...).Result()
	if err != nil {
		return
	}

	cities = make([]models.City, 0, len(redisData))
	for _, data := range redisData {
		var city models.City
		err = json.Unmarshal([]byte(data.(string)), &city)
		if err != nil {
			return
		}

		cities = append(cities, city)
	}

	return cities, nil
}

func (c *CinemaCache) GetHallConfiguraion(ctx context.Context, id int32) (places []models.Place, err error) {
	defer c.updateMetrics(&err, "GetHallConfiguraion")
	defer handleError(ctx, &err)
	defer c.logError(&err, "GetHallConfiguraion")
	data, err := c.hallsConfigurationsRdb.Get(ctx, fmt.Sprint(id)).Bytes()
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &places)
	if err != nil {
		return
	}

	return places, nil
}

func (c *CinemaCache) CacheCinemasInCity(ctx context.Context, id int32, cinemas []models.Cinema, ttl time.Duration) (err error) {
	defer handleError(ctx, &err)
	defer c.logError(&err, "CacheCinemasInCity")
	data, err := json.Marshal(cinemas)
	if err != nil {
		return
	}

	err = c.citiesCinemasRdb.Set(ctx, fmt.Sprint(id), data, ttl).Err()
	return
}

func (c *CinemaCache) CacheCinemasCities(ctx context.Context, cities []models.City, ttl time.Duration) (err error) {
	defer handleError(ctx, &err)
	defer c.logError(&err, "CacheCinemasCities")
	tx := c.citiesRdb.Pipeline()
	for _, city := range cities {
		toCache, merr := json.Marshal(city)
		if merr != nil {
			err = merr
			return
		}
		tx.Set(ctx, fmt.Sprint(city.ID), toCache, ttl)
	}
	_, err = tx.Exec(ctx)
	return
}

func (c *CinemaCache) CacheHallConfiguraion(ctx context.Context, id int32, places []models.Place, ttl time.Duration) (err error) {
	defer handleError(ctx, &err)
	defer c.logError(&err, "CacheHallConfiguraion")
	toCache, err := json.Marshal(places)
	if err != nil {
		return
	}

	err = c.hallsConfigurationsRdb.Set(ctx, fmt.Sprint(id), toCache, ttl).Err()
	return
}

func (c *CinemaCache) CacheHalls(ctx context.Context, halls []models.Hall, ttl time.Duration) (err error) {
	defer handleError(ctx, &err)
	defer c.logError(&err, "CacheHalls")
	tx := c.hallsRdb.Pipeline()
	for _, hall := range halls {
		toCache, merr := json.Marshal(hall)
		if merr != nil {
			err = merr
			return
		}
		err = tx.Set(ctx, fmt.Sprint(hall.ID), toCache, ttl).Err()
		if err != nil {
			return
		}
	}
	_, err = tx.Exec(ctx)
	return
}

func (c *CinemaCache) GetHalls(ctx context.Context, ids []int32) (halls []models.Hall, notFoundedIds []int32, err error) {
	defer c.updateMetrics(&err, "GetHalls")
	defer handleError(ctx, &err)
	defer c.logError(&err, "GetHalls")
	keys := make([]string, len(ids))
	hallsIds := make(map[int32]struct{}, len(ids))
	for i, id := range ids {
		keys[i] = fmt.Sprint(id)
		hallsIds[id] = struct{}{}
	}

	hallsBody, err := c.hallsRdb.MGet(ctx, keys...).Result()
	if err != nil {
		return
	}

	halls = make([]models.Hall, 0, len(hallsBody))
	for _, cached := range hallsBody {
		if cached == nil {
			continue
		}

		hall := models.Hall{}
		err = json.Unmarshal([]byte(cached.(string)), &hall)
		if err != nil {
			return
		}
		delete(hallsIds, hall.ID)
		halls = append(halls, hall)
	}

	return halls, maps.Keys(hallsIds), nil
}

func (c *CinemaCache) logError(errptr *error, functionName string) {
	if errptr == nil || *errptr == nil {
		return
	}

	err := *errptr
	var repoErr = &models.ServiceError{}
	if errors.As(err, &repoErr) {
		c.logger.WithFields(
			logrus.Fields{
				"error.function.name": functionName,
				"error.msg":           repoErr.Msg,
				"error.code":          repoErr.Code,
			},
		).Error("casts cache error occurred")
	} else {
		c.logger.WithFields(
			logrus.Fields{
				"error.function.name": functionName,
				"error.msg":           err.Error(),
			},
		).Error("casts cache error occurred")
	}
}

func handleError(ctx context.Context, err *error) {
	if ctx.Err() != nil {
		var code models.ErrorCode
		switch {
		case errors.Is(ctx.Err(), context.Canceled):
			code = models.Canceled
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			code = models.DeadlineExceeded
		}
		*err = models.Error(code, ctx.Err().Error())
		return
	}

	if err == nil || *err == nil {
		return
	}

	var repoErr = &models.ServiceError{}
	if !errors.As(*err, &repoErr) {
		var code models.ErrorCode
		switch {
		case errors.Is(*err, redis.Nil):
			code = models.NotFound
			*err = models.Error(code, "entity not found")
		default:
			code = models.Internal
			*err = models.Error(code, "cache internal error")
		}
	}
}

func (c *CinemaCache) updateMetrics(err *error, functionName string) {
	if err == nil || *err == nil {
		c.metrics.IncCacheHits(functionName, 1)
		return
	}
	if models.Code(*err) == models.NotFound {
		c.metrics.IncCacheMiss(functionName, 1)
	}
}
