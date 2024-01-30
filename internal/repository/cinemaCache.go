package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
)

type cinemaCache struct {
	logger                 *logrus.Logger
	cinemasRdb             *redis.Client
	citiesRdb              *redis.Client
	hallsConfigurationsRdb *redis.Client
	hallsRdb               *redis.Client
}

func NewCinemaCache(logger *logrus.Logger, cinemasRdb, citiesRdb, hallsConfigurationsRdb, hallsRdb *redis.Client) *cinemaCache {
	return &cinemaCache{
		logger:                 logger,
		cinemasRdb:             cinemasRdb,
		citiesRdb:              citiesRdb,
		hallsConfigurationsRdb: hallsConfigurationsRdb,
		hallsRdb:               hallsRdb,
	}
}

func (c *cinemaCache) PingContext(ctx context.Context) error {
	if err := c.cinemasRdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("error while pinging cinema cache: %w", err)
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

func (c *cinemaCache) GetCinemasInCity(ctx context.Context, id int32) ([]Cinema, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaCache.GetCinemasInCity")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	data, err := c.cinemasRdb.Get(ctx, fmt.Sprint(id)).Bytes()
	if err != nil {
		c.logger.Errorf("error while getting cinemas in city %s", err.Error())
		return []Cinema{}, err
	}

	var cinemas []Cinema
	err = json.Unmarshal(data, &cinemas)
	if err != nil {
		c.logger.Errorf("error while unmarchalling cinemas in city %s", err.Error())
		return []Cinema{}, err
	}

	return cinemas, nil
}

func (c *cinemaCache) GetCinemasCities(ctx context.Context) ([]City, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaCache.CacheCinemasInCity")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	var keys []string
	c.citiesRdb.Keys(ctx, "*").ScanSlice(&keys)
	if err != nil {
		c.logger.Errorf("error while getting cinemas cities keys %s", err.Error())
		return []City{}, err
	}

	redisData, err := c.citiesRdb.MGet(ctx, keys...).Result()
	if err != nil {
		c.logger.Errorf("error while getting cinemas cities %s", err.Error())
		return []City{}, err
	}

	var cities = make([]City, 0, len(redisData))
	for _, data := range redisData {
		var city City
		err = json.Unmarshal([]byte(data.(string)), &city)
		if err != nil {
			c.logger.Errorf("error while unmarchalling cinemas in city %s", err.Error())
			return []City{}, err
		}

		cities = append(cities, city)
	}

	return cities, nil
}

func (c *cinemaCache) GetHallConfiguraion(ctx context.Context, id int32) ([]Place, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaCache.GetCinemasInCity")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	data, err := c.hallsConfigurationsRdb.Get(ctx, fmt.Sprint(id)).Bytes()
	if err != nil {
		c.logger.Errorf("error while getting places %s", err.Error())
		return []Place{}, err
	}

	var places []Place
	err = json.Unmarshal(data, &places)
	if err != nil {
		c.logger.Errorf("error while unmarchalling places %s", err.Error())
		return []Place{}, err
	}

	return places, nil
}

func (c *cinemaCache) CacheCinemasInCity(ctx context.Context, id int32, cinemas []Cinema, ttl time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaCache.CacheCinemasInCity")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	data, err := json.Marshal(cinemas)
	if err != nil {
		return err
	}

	err = c.cinemasRdb.Set(ctx, fmt.Sprint(id), data, ttl).Err()
	return err
}

func (c *cinemaCache) CacheCinemasCities(ctx context.Context, cities []City, ttl time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaCache.CacheCinemasCities")
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

func (c *cinemaCache) CacheHallConfiguraion(ctx context.Context, id int32, places []Place, ttl time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaCache.CacheHallConfiguraion")
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

func (c *cinemaCache) CacheHalls(ctx context.Context, halls []Hall, ttl time.Duration) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaCache.CacheHalls")
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

func (c *cinemaCache) GetHalls(ctx context.Context, ids []int32) ([]Hall, []int32, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaCache.GetHalls")
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
		return []Hall{}, []int32{}, err
	}

	var cachedHalls = make([]Hall, 0, len(halls))
	for _, cached := range halls {
		if cached == nil {
			continue
		}

		hall := Hall{}
		err = json.Unmarshal([]byte(cached.(string)), &hall)
		if err != nil {
			return []Hall{}, []int32{}, err
		}
		delete(hallsIds, hall.Id)
		cachedHalls = append(cachedHalls, hall)
	}

	return cachedHalls, maps.Keys(hallsIds), nil
}
