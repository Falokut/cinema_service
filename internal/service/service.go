package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	cinema_service "github.com/Falokut/cinema_service/pkg/cinema_service/v1/protos"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CinemaRepository interface {
	GetCinemasInCity(ctx context.Context, id int32) (*cinema_service.Cinemas, error)

	GetAllMoviesScreenings(ctx context.Context, startPeriod, endPeriod time.Time) (*cinema_service.PreviewScreenings, error)
	GetMoviesScreeningsInCities(ctx context.Context, citiesIds []int32,
		startPeriod, endPeriod time.Time) (*cinema_service.PreviewScreenings, error)
	GetMoviesScreenings(ctx context.Context, cinemaID int32,
		startPeriod, endPeriod time.Time) (*cinema_service.PreviewScreenings, error)
	GetScreenings(ctx context.Context, cinemaID, movieID int32,
		startPeriod, endPeriod time.Time) (*cinema_service.Screenings, error)
	GetCityScreenings(ctx context.Context, cityID, movieID int32,
		startPeriod, endPeriod time.Time) (*cinema_service.CityScreenings, error)
	GetCinemasCities(ctx context.Context) (*cinema_service.Cities, error)
	GetCinema(ctx context.Context, id int32) (*cinema_service.Cinema, error)
	GetHallConfiguraion(ctx context.Context, id int32) (*cinema_service.HallConfiguration, error)
	GetHalls(ctx context.Context, ids []int32) (*cinema_service.Halls, error)

	GetScreening(ctx context.Context, id int32) (*cinema_service.GetScreeningResponse, error)
}

type CinemaService struct {
	cinema_service.UnimplementedCinemaServiceV1Server
	logger       *logrus.Logger
	cinemaRepo   CinemaRepository
	errorHandler errorHandler
}

func NewCinemaService(logger *logrus.Logger, cinemaRepo CinemaRepository) *CinemaService {
	errorHandler := newErrorHandler(logger)
	return &CinemaService{logger: logger, errorHandler: errorHandler, cinemaRepo: cinemaRepo}
}

func (s *CinemaService) GetCinemasInCity(ctx context.Context,
	in *cinema_service.GetCinemasInCityRequest) (*cinema_service.Cinemas, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaService.GetCinemasInCity")
	defer span.Finish()

	res, err := s.cinemaRepo.GetCinemasInCity(ctx, in.CityId)
	if err != nil {
		ext.LogError(span, err)
		span.SetTag("grpc.status", status.Code(err))
		return nil, err
	}
	if len(res.Cinemas) == 0 {
		return nil, s.errorHandler.createErrorResponceWithSpan(span, ErrNotFound,
			fmt.Sprintf("no cinema found in city with id %d", in.CityId))
	}

	span.SetTag("grpc.status", codes.OK)
	return res, nil
}

func (s *CinemaService) GetMoviesScreenings(ctx context.Context,
	in *cinema_service.GetMoviesScreeningsRequest) (*cinema_service.PreviewScreenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaService.GetMoviesScreenings")
	defer span.Finish()

	start, end, err := parsePeriods(in.StartPeriod, in.EndPeriod)
	if err != nil {
		return nil, s.errorHandler.createErrorResponceWithSpan(span, ErrInvalidArgument, err.Error())
	}
	res, err := s.cinemaRepo.GetMoviesScreenings(ctx, in.CinemaId, start, end)
	if err != nil {
		ext.LogError(span, err)
		span.SetTag("grpc.status", status.Code(err))
		return nil, err
	}

	span.SetTag("grpc.status", codes.OK)
	return res, nil
}

func (s *CinemaService) GetMoviesScreeningsInCities(ctx context.Context,
	in *cinema_service.GetMoviesScreeningsInCitiesRequest) (*cinema_service.PreviewScreenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaService.GetMoviesScreeningsInCities")
	defer span.Finish()

	start, end, err := parsePeriods(in.StartPeriod, in.EndPeriod)
	if err != nil {
		return nil, s.errorHandler.createErrorResponceWithSpan(span, ErrInvalidArgument, err.Error())
	}

	var res *cinema_service.PreviewScreenings
	if in.CitiesIds == nil {
		res, err = s.cinemaRepo.GetAllMoviesScreenings(ctx, start, end)
	} else {
		if err := checkIds(in.GetCitiesIds()); err != nil {
			return nil, s.errorHandler.createErrorResponceWithSpan(span, ErrInvalidArgument, "invalid ")
		}
		ids := convertStringsSlice(strings.Split(in.GetCitiesIds(), ","))
		res, err = s.cinemaRepo.GetMoviesScreeningsInCities(ctx, ids, start, end)
	}

	if err != nil {
		ext.LogError(span, err)
		span.SetTag("grpc.status", status.Code(err))
		return nil, err
	}

	span.SetTag("grpc.status", codes.OK)
	return res, nil
}

func convertStringsSlice(str []string) []int32 {
	var nums = make([]int32, 0, len(str))
	for _, s := range str {
		num, err := strconv.Atoi(s)
		if err == nil {
			nums = append(nums, int32(num))
		}
	}
	return nums
}

func (s *CinemaService) GetScreenings(ctx context.Context,
	in *cinema_service.GetScreeningsRequest) (*cinema_service.Screenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaService.GetScreenings")
	defer span.Finish()
	start, end, err := parsePeriods(in.StartPeriod, in.EndPeriod)
	if err != nil {
		return nil, s.errorHandler.createErrorResponceWithSpan(span, ErrInvalidArgument, err.Error())
	}

	res, err := s.cinemaRepo.GetScreenings(ctx, in.CinemaId, in.MovieID, start, end)
	if err != nil {
		ext.LogError(span, err)
		span.SetTag("grpc.status", status.Code(err))
		return nil, err
	}

	span.SetTag("grpc.status", codes.OK)
	return res, nil
}

func (s *CinemaService) GetScreeningsInCity(ctx context.Context,
	in *cinema_service.GetScreeningsInCityRequest) (*cinema_service.CityScreenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaService.GetScreeningsInCity")
	defer span.Finish()
	start, end, err := parsePeriods(in.StartPeriod, in.EndPeriod)
	if err != nil {
		return nil, s.errorHandler.createErrorResponceWithSpan(span, ErrInvalidArgument, err.Error())
	}

	res, err := s.cinemaRepo.GetCityScreenings(ctx, in.CityId, in.MovieId, start, end)
	if err != nil {
		ext.LogError(span, err)
		span.SetTag("grpc.status", status.Code(err))
		return nil, err
	}

	span.SetTag("grpc.status", codes.OK)
	return res, nil
}

func (s *CinemaService) GetScreening(ctx context.Context,
	in *cinema_service.GetScreeningRequest) (*cinema_service.GetScreeningResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaService.GetScreening")
	defer span.Finish()

	res, err := s.cinemaRepo.GetScreening(ctx, in.ScreeningId)
	if err != nil {
		ext.LogError(span, err)
		span.SetTag("grpc.status", status.Code(err))
		return nil, err
	}

	span.SetTag("grpc.status", codes.OK)
	return res, nil
}

func (s *CinemaService) GetCinemasCities(ctx context.Context,
	in *emptypb.Empty) (*cinema_service.Cities, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaService.GetCinemasCities")
	defer span.Finish()

	res, err := s.cinemaRepo.GetCinemasCities(ctx)
	if err != nil {
		ext.LogError(span, err)
		span.SetTag("grpc.status", status.Code(err))
		return nil, err
	}
	span.SetTag("grpc.status", codes.OK)
	return res, nil
}

func (s *CinemaService) GetHallConfiguration(ctx context.Context,
	in *cinema_service.GetHallConfigurationRequest) (*cinema_service.HallConfiguration, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaService.GetHallConfiguration")
	defer span.Finish()
	res, err := s.cinemaRepo.GetHallConfiguraion(ctx, in.HallId)
	if err != nil {
		ext.LogError(span, err)
		span.SetTag("grpc.status", status.Code(err))
		return nil, err
	}
	span.SetTag("grpc.status", codes.OK)
	return res, nil
}

func (s *CinemaService) GetCinema(ctx context.Context,
	in *cinema_service.GetCinemaRequest) (*cinema_service.Cinema, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaService.GetCinema")
	defer span.Finish()

	cinema, err := s.cinemaRepo.GetCinema(ctx, in.CinemaId)
	if err != nil {
		ext.LogError(span, err)
		span.SetTag("grpc.status", status.Code(err))
		return nil, err
	}

	span.SetTag("grpc.status", codes.OK)
	return cinema, nil
}
func (s *CinemaService) GetHalls(ctx context.Context,
	in *cinema_service.GetHallsRequest) (*cinema_service.Halls, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaService.GetHalls")
	defer span.Finish()

	in.HallsIds = strings.ReplaceAll(in.HallsIds, `"`, "")
	if err := checkIds(in.HallsIds); err != nil {
		return nil, s.errorHandler.createErrorResponceWithSpan(span, ErrInvalidArgument, err.Error())
	}

	ids := convertStringsSlice(strings.Split(in.HallsIds, ","))
	if len(ids) == 0 {
		return nil, s.errorHandler.createErrorResponceWithSpan(span, ErrInvalidArgument, "halls_ids musn't be empty")
	}

	res, err := s.cinemaRepo.GetHalls(ctx, ids)
	if err != nil {
		ext.LogError(span, err)
		span.SetTag("grpc.status", status.Code(err))
		return nil, err
	}
	if len(res.Halls) == 0 {
		return nil, s.errorHandler.createErrorResponceWithSpan(span, ErrNotFound, fmt.Sprintf("halls with ids %s not found", in.HallsIds))
	}
	span.SetTag("grpc.status", codes.OK)
	return res, nil
}

func parsePeriods(startPeriod, endPeriod *cinema_service.Timestamp) (start time.Time, end time.Time, err error) {
	if startPeriod == nil || endPeriod == nil {
		err = fmt.Errorf("invalid period value, it mustn't be empty")
		return
	}
	start, err = time.Parse(time.RFC3339, startPeriod.FormattedTimestamp)
	if err != nil {
		err = fmt.Errorf("invalid start period value, it must be RFC3339 layout value: %s", startPeriod)
		return
	}
	end, err = time.Parse(time.RFC3339, endPeriod.FormattedTimestamp)
	if err != nil {
		err = fmt.Errorf("invalid start period value, it must be RFC3339 layout value: %s", endPeriod)
		return
	}

	return
}

func checkIds(val string) error {
	exp := regexp.MustCompile("^[!-&!+,0-9]+$")

	if !exp.Match([]byte(val)) {
		return ErrInvalidArgument
	}

	return nil
}
