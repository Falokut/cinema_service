package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Falokut/cinema_service/internal/repository"
	cinema_service "github.com/Falokut/cinema_service/pkg/cinema_service/v1/protos"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

type CacheConfig struct {
	HallConfigurationTTL time.Duration
	CinemasTTL           time.Duration
	CitiesTTL            time.Duration
}

type Metrics interface {
	IncCacheMiss(method string, num int)
	IncCacheHits(method string, num int)
}

type cinemaRepositoryWrapper struct {
	errorHandler
	logger *logrus.Logger
	repo   repository.CinemaRepository
	cache  repository.CinemaCache

	cacheCfg CacheConfig
	metrics  Metrics
}

func NewCinemaRepositoryWrapper(logger *logrus.Logger, cinemaRepository repository.CinemaRepository,
	cache repository.CinemaCache, cacheCfg CacheConfig, metrics Metrics) *cinemaRepositoryWrapper {
	errorHandler := newErrorHandler(logger)
	return &cinemaRepositoryWrapper{
		logger:       logger,
		errorHandler: errorHandler,
		repo:         cinemaRepository,
		cache:        cache,
		cacheCfg:     cacheCfg,
		metrics:      metrics,
	}
}

func (w *cinemaRepositoryWrapper) GetCinemasInCity(ctx context.Context, id int32) (*cinema_service.Cinemas, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaRepositoryWrapper.GetCinemasInCity")
	defer span.Finish()

	cinemas, err := w.cache.GetCinemasInCity(ctx, id)
	if err == nil {
		w.metrics.IncCacheHits("GetCinemasInCity", 1)
		return convertToCinemas(cinemas), nil
	}
	w.metrics.IncCacheMiss("GetCinemasInCity", 1)

	cinemas, err = w.repo.GetCinemasInCity(ctx, id)
	if err != nil {
		return nil, err
	}

	go func() {
		err := w.cache.CacheCinemasInCity(context.Background(), id, cinemas, w.cacheCfg.CinemasTTL)
		if err != nil {
			w.logger.Errorf("error while caching hall configuration, %s", err)
		}
	}()

	return convertToCinemas(cinemas), nil
}

func convertToCinemas(cinemas []repository.Cinema) *cinema_service.Cinemas {
	res := &cinema_service.Cinemas{}
	res.Cinemas = make([]*cinema_service.Cinema, len(cinemas))
	for i, cinema := range cinemas {
		res.Cinemas[i] = &cinema_service.Cinema{
			CinemaID: cinema.ID,
			Name:     cinema.Name,
			Address:  cinema.Address,
			Coordinates: &cinema_service.Coordinates{
				Latityde:  cinema.Coordinates.Latityde,
				Longitude: cinema.Coordinates.Longitude,
			},
		}
	}
	return res
}

func (w *cinemaRepositoryWrapper) GetMoviesScreenings(ctx context.Context, cinemaID int32,
	startPeriod, endPeriod time.Time) (*cinema_service.PreviewScreenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx,
		"cinemaRepositoryWrapper.GetMoviesScreenings")
	defer span.Finish()

	previews, err := w.repo.GetMoviesScreenings(ctx, cinemaID, startPeriod, endPeriod)
	if err != nil {
		return nil, w.createErrorResponceWithSpan(span, ErrInternal, err.Error())
	}
	if len(previews) == 0 {
		return nil, w.createErrorResponceWithSpan(span, ErrNotFound, "")
	}

	res := &cinema_service.PreviewScreenings{}
	res.Screenings = make([]*cinema_service.PreviewScreening, len(previews))
	for i, preview := range previews {
		res.Screenings[i] = &cinema_service.PreviewScreening{
			MovieID:         preview.MovieID,
			ScreeningsTypes: preview.ScreeningsTypes,
			HallsTypes:      preview.HallsTypes,
		}
	}

	return res, nil
}

func (w *cinemaRepositoryWrapper) GetScreenings(ctx context.Context, cinemaID, movieID int32,
	startPeriod, endPeriod time.Time) (*cinema_service.Screenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx,
		"cinemaRepositoryWrapper.GetScreenings")
	defer span.Finish()

	screenings, err := w.repo.GetScreenings(ctx, cinemaID, movieID,
		startPeriod, endPeriod)
	if err != nil {
		return nil, w.createErrorResponceWithSpan(span, ErrInternal, err.Error())
	}
	if len(screenings) == 0 {
		return nil, w.createErrorResponceWithSpan(span, ErrNotFound, "")
	}

	res := &cinema_service.Screenings{}
	res.Screenings = make([]*cinema_service.Screening, len(screenings))
	for i, screening := range screenings {
		res.Screenings[i] = &cinema_service.Screening{
			ScreeningID:   screening.ScreeningID,
			ScreeningType: screening.ScreeningType,
			MovieID:       screening.MovieID,
			HallID:        screening.HallID,
			StartTime:     &cinema_service.Timestamp{FormattedTimestamp: screening.StartTime.Format(time.RFC3339)},
			TicketPrice:   decimalFromFloat(screening.TicketPrice),
		}
	}

	return res, nil
}

func (w *cinemaRepositoryWrapper) GetCinemasCities(ctx context.Context) (*cinema_service.Cities, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx,
		"cinemaRepositoryWrapper.GetCinemasCities")
	defer span.Finish()

	cities, err := w.cache.GetCinemasCities(ctx)
	if err == nil {
		w.metrics.IncCacheHits("GetCinemasCities", 1)
		return convertToCities(cities), nil
	}
	w.metrics.IncCacheMiss("GetCinemasCities", 1)

	cities, err = w.repo.GetCinemasCities(ctx)
	if err != nil {
		return nil, w.createErrorResponceWithSpan(span, ErrInternal, err.Error())
	}
	if len(cities) == 0 {
		return nil, w.createErrorResponceWithSpan(span, ErrNotFound, "")
	}

	go func() {
		err := w.cache.CacheCinemasCities(context.Background(), cities, w.cacheCfg.CitiesTTL)
		if err != nil {
			w.logger.Errorf("error while caching cinemas cities, %s", err)
		}
	}()

	return convertToCities(cities), nil
}

func convertToCities(cities []repository.City) *cinema_service.Cities {
	res := &cinema_service.Cities{}
	res.Cities = make([]*cinema_service.City, len(cities))
	for i, city := range cities {
		res.Cities[i] = &cinema_service.City{CityID: city.ID, Name: city.Name}
	}
	return res
}

func (w *cinemaRepositoryWrapper) GetHallConfiguraion(ctx context.Context,
	id int32) (*cinema_service.HallConfiguration, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx,
		"cinemaRepositoryWrapper.GetHallConfiguraion")
	defer span.Finish()

	places, err := w.cache.GetHallConfiguraion(ctx, id)
	if err == nil {
		w.metrics.IncCacheHits("GetHallConfiguraion", 1)
		return convertToHallConfiguration(places), nil
	}
	w.metrics.IncCacheMiss("GetHallConfiguraion", 1)

	places, err = w.repo.GetHallConfiguraion(ctx, id)
	if err != nil {
		return nil, w.createErrorResponceWithSpan(span, ErrInternal, err.Error())
	}
	if len(places) == 0 {
		return nil, w.createErrorResponceWithSpan(span, ErrNotFound, "")
	}

	go func() {
		err := w.cache.CacheHallConfiguraion(context.Background(), id, places, w.cacheCfg.HallConfigurationTTL)
		if err != nil {
			w.logger.Errorf("error while caching hall configuration, %s", err)
		}
	}()
	return convertToHallConfiguration(places), nil
}

func convertToHallConfiguration(places []repository.Place) *cinema_service.HallConfiguration {
	res := &cinema_service.HallConfiguration{}
	res.Place = make([]*cinema_service.Place, len(places))
	for i, place := range places {
		res.Place[i] = &cinema_service.Place{
			Row:      place.Row,
			Seat:     place.Seat,
			GridPosX: place.GridPosX,
			GridPosY: place.GridPosY,
		}
	}
	return res
}
func decimalFromFloat(n string) *cinema_service.DecimalValue {
	nums := strings.Split(n, ".")
	if len(nums) != 2 {
		return nil
	}
	units, _ := strconv.ParseInt(nums[0], 10, 64)
	nanos, _ := strconv.ParseInt(nums[1], 10, 32)
	return &cinema_service.DecimalValue{
		Units: units,
		Nanos: int32(nanos),
	}
}
