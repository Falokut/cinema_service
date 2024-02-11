package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Falokut/cinema_service/internal/repository"
	cinema_service "github.com/Falokut/cinema_service/pkg/cinema_service/v1/protos"
	"github.com/opentracing/opentracing-go"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

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
		err := w.cache.CacheCinemasInCity(context.Background(), id, cinemas, w.cacheCfg.CitiesCinemasTTL)
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

	return convertMoviesScreeningsToProto(previews), nil
}

func (w *cinemaRepositoryWrapper) GetAllMoviesScreenings(ctx context.Context,
	startPeriod, endPeriod time.Time) (*cinema_service.PreviewScreenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx,
		"cinemaRepositoryWrapper.GetAllMoviesScreenings")
	defer span.Finish()

	previews, err := w.repo.GetAllMoviesScreenings(ctx, startPeriod, endPeriod)
	if err != nil {
		return nil, w.createErrorResponceWithSpan(span, ErrInternal, err.Error())
	}
	if len(previews) == 0 {
		return nil, w.createErrorResponceWithSpan(span, ErrNotFound, "")
	}

	return convertMoviesScreeningsToProto(previews), nil
}

func (w *cinemaRepositoryWrapper) GetMoviesScreeningsInCities(ctx context.Context, citiesIds []int32,
	startPeriod, endPeriod time.Time) (*cinema_service.PreviewScreenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx,
		"cinemaRepositoryWrapper.GetMoviesScreeningsInCities")
	defer span.Finish()

	previews, err := w.repo.GetMoviesScreeningsInCities(ctx, citiesIds,
		startPeriod, endPeriod)
	if err != nil {
		return nil, w.createErrorResponceWithSpan(span, ErrInternal, err.Error())
	}
	if len(previews) == 0 {
		return nil, w.createErrorResponceWithSpan(span, ErrNotFound, "")
	}

	return convertMoviesScreeningsToProto(previews), nil
}

func convertMoviesScreeningsToProto(previews []repository.MoviesScreenings) *cinema_service.PreviewScreenings {
	res := &cinema_service.PreviewScreenings{}
	res.Screenings = make([]*cinema_service.PreviewScreening, len(previews))
	for i, preview := range previews {
		res.Screenings[i] = &cinema_service.PreviewScreening{
			MovieID:         preview.MovieID,
			ScreeningsTypes: preview.ScreeningsTypes,
			HallsTypes:      preview.HallsTypes,
		}
	}
	return res
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
	if len(screenings.Screenings) == 0 {
		return nil, w.createErrorResponceWithSpan(span, ErrNotFound, "")
	}

	res := &cinema_service.Screenings{}
	res.Screenings = make([]*cinema_service.Screening, len(screenings.Screenings))
	for i, screening := range screenings.Screenings {
		res.Screenings[i] = &cinema_service.Screening{
			ScreeningID:   screening.ScreeningID,
			ScreeningType: screening.ScreeningType,
			MovieID:       screening.MovieID,
			HallID:        screening.HallID,
			StartTime:     &cinema_service.Timestamp{FormattedTimestamp: screening.StartTime.Format(time.RFC3339)},
			TicketPrice:   priceFromFloat(screening.TicketPrice),
		}
	}

	return res, nil
}

func (w *cinemaRepositoryWrapper) GetCityScreenings(ctx context.Context, cityID, movieID int32,
	startPeriod, endPeriod time.Time) (*cinema_service.CityScreenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx,
		"cinemaRepositoryWrapper.GetCityScreenings")
	defer span.Finish()

	screenings, err := w.repo.GetCityScreenings(ctx, cityID, movieID,
		startPeriod, endPeriod)
	if err != nil {
		return nil, w.createErrorResponceWithSpan(span, ErrInternal, err.Error())
	}
	if len(screenings) == 0 {
		return nil, w.createErrorResponceWithSpan(span, ErrNotFound, "")
	}

	res := &cinema_service.CityScreenings{}
	res.Screenings = make([]*cinema_service.CityScreening, len(screenings))
	for i, screening := range screenings {
		res.Screenings[i] = &cinema_service.CityScreening{
			ScreeningId:   screening.ScreeningId,
			CinemaId:      screening.CinemaId,
			ScreeningType: screening.ScreeningType,
			HallID:        screening.HallId,
			StartTime:     &cinema_service.Timestamp{FormattedTimestamp: screening.StartTime.Format(time.RFC3339)},
			TicketPrice:   priceFromFloat(screening.TicketPrice),
		}
	}

	return res, nil
}

func (w *cinemaRepositoryWrapper) GetScreening(ctx context.Context, id int32) (*cinema_service.GetScreeningResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaRepositoryWrapper.GetScreening")
	defer span.Finish()

	screening, err := w.repo.GetScreening(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, w.createErrorResponceWithSpan(span, ErrNotFound, fmt.Sprintf("screening with %d id not found", id))
	}
	if err != nil {
		return nil, w.createErrorResponceWithSpan(span, ErrInternal, err.Error())
	}

	res := &cinema_service.GetScreeningResponse{
		ScreeningType: screening.ScreeningType,
		CinemaId:      screening.CinemaId,
		MovieId:       screening.MovieId,
		HallID:        screening.HallId,
		StartTime:     &cinema_service.Timestamp{FormattedTimestamp: screening.StartTime.Format(time.RFC3339)},
		TicketPrice:   priceFromFloat(screening.TicketPrice),
	}

	res.HallConfiguration, err = w.GetHallConfiguraion(ctx, res.HallID)
	if err != nil {
		return nil, w.createErrorResponceWithSpan(span, ErrInternal, err.Error())
	}
	return res, nil
}

func (w *cinemaRepositoryWrapper) GetCinema(ctx context.Context, id int32) (*cinema_service.Cinema, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaRepositoryWrapper.GetCinema")
	defer span.Finish()

	cinema, err := w.cache.GetCinema(ctx, id)
	if err == nil {
		w.metrics.IncCacheHits("GetCinema", 1)
		return convertToCinema(cinema), nil
	}
	w.metrics.IncCacheMiss("GetCinema", 1)

	cinema, err = w.repo.GetCinema(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, w.createErrorResponceWithSpan(span, ErrNotFound, "")
	}
	if err != nil {
		return nil, w.createErrorResponceWithSpan(span, ErrInternal, err.Error())
	}

	go func() {
		err := w.cache.CacheCinema(context.Background(), cinema, w.cacheCfg.CinemasTTL)
		if err != nil {
			w.logger.Errorf("error while cinema, %s", err)
		}
	}()
	return convertToCinema(cinema), nil
}

func convertToCinema(cinema repository.Cinema) *cinema_service.Cinema {
	return &cinema_service.Cinema{
		CinemaID: cinema.ID,
		Name:     cinema.Name,
		Address:  cinema.Address,
		Coordinates: &cinema_service.Coordinates{
			Latityde:  cinema.Coordinates.Latityde,
			Longitude: cinema.Coordinates.Longitude,
		},
	}
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

func (w *cinemaRepositoryWrapper) GetHalls(ctx context.Context,
	ids []int32) (*cinema_service.Halls, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx,
		"cinemaRepositoryWrapper.GetHalls")
	defer span.Finish()
	w.logger.Info("Searching halls in cache")
	cachedHalls, notFoundedIds, err := w.cache.GetHalls(ctx, ids)
	if errors.Is(err, redis.Nil) {
		w.metrics.IncCacheMiss("GetHalls", len(ids))
	} else if err != nil {
		w.logger.Error(err)
	}
	w.logger.Debugf("Not found halls ids in cache: %v", notFoundedIds)

	if len(cachedHalls) == len(ids) {
		w.metrics.IncCacheHits("GetHalls", len(ids))
		return convertToHalls(cachedHalls), nil
	}

	if len(cachedHalls) != 0 && err == nil {
		w.metrics.IncCacheHits("GetHalls", len(ids)-len(notFoundedIds))
		w.metrics.IncCacheMiss("GetHalls", len(notFoundedIds))
		ids = notFoundedIds
	}

	w.logger.Info("Searching halls in repository")
	halls, err := w.repo.GetHalls(ctx, notFoundedIds)
	if err != nil {
		return nil, w.createErrorResponceWithSpan(span, ErrInternal, err.Error())
	}
	halls = append(halls, cachedHalls...)

	go func() {
		err := w.cache.CacheHalls(context.Background(), halls, w.cacheCfg.HallsTTL)
		if err != nil {
			w.logger.Errorf("error while caching halls, %s", err)
		}
	}()
	return convertToHalls(halls), nil
}

func convertToHalls(halls []repository.Hall) *cinema_service.Halls {
	res := &cinema_service.Halls{}
	res.Halls = make([]*cinema_service.Hall, len(halls))
	for i, hall := range halls {
		res.Halls[i] = &cinema_service.Hall{
			HallID:   hall.Id,
			Name:     hall.Name,
			Type:     hall.Type,
			HallSize: hall.Size,
		}
	}

	return res
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

func priceFromFloat(n string) *cinema_service.Price {
	nums := strings.Split(n, ".")
	if len(nums) != 2 {
		return nil
	}
	units, _ := strconv.ParseInt(nums[0], 10, 64)
	nanos, _ := strconv.ParseInt(nums[1], 10, 32)
	return &cinema_service.Price{Value: int32(units)*100 + int32(nanos)}
}
