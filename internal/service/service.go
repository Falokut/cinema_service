package service

import (
	"context"
	"fmt"
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

	GetMoviesScreenings(ctx context.Context, cinemaID int32,
		startPeriod, endPeriod time.Time) (*cinema_service.PreviewScreenings, error)
	GetScreenings(ctx context.Context, cinemaID, movieID int32,
		startPeriod, endPeriod time.Time) (*cinema_service.Screenings, error)
	GetCinemasCities(ctx context.Context) (*cinema_service.Cities, error)
	GetHallConfiguraion(ctx context.Context, id int32) (*cinema_service.HallConfiguration, error)
}

type cinemaService struct {
	cinema_service.UnimplementedCinemaServiceV1Server
	logger       *logrus.Logger
	cinemaRepo   CinemaRepository
	errorHandler errorHandler
}

func NewCinemaService(logger *logrus.Logger, cinemaRepo CinemaRepository) *cinemaService {
	errorHandler := newErrorHandler(logger)
	return &cinemaService{logger: logger, errorHandler: errorHandler, cinemaRepo: cinemaRepo}
}

func (s *cinemaService) GetCinemasInCity(ctx context.Context,
	in *cinema_service.GetCinemasInCityRequest) (*cinema_service.Cinemas, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaService.GetCinemasInCity")
	defer span.Finish()

	res, err := s.cinemaRepo.GetCinemasInCity(ctx, in.CityId)
	if err != nil {
		ext.LogError(span, err)
		span.SetTag("grpc.status", status.Code(err))
		return nil, err
	}

	span.SetTag("grpc.status", codes.OK)
	return res, nil
}

func (s *cinemaService) GetMoviesScreenings(ctx context.Context,
	in *cinema_service.GetMoviesScreeningsRequest) (*cinema_service.PreviewScreenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaService.GetMoviesScreenings")
	defer span.Finish()

	start, end, err := parsePeriods(in.StartPeriod.FormattedTimestamp, in.EndPeriod.FormattedTimestamp)
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

func (s *cinemaService) GetScreenings(ctx context.Context,
	in *cinema_service.GetScreeningsRequest) (*cinema_service.Screenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaService.GetScreenings")
	defer span.Finish()
	start, end, err := parsePeriods(in.StartPeriod.FormattedTimestamp, in.EndPeriod.FormattedTimestamp)
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

func (s *cinemaService) GetCinemasCities(ctx context.Context,
	in *emptypb.Empty) (*cinema_service.Cities, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaService.GetCinemasCities")
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

func (s *cinemaService) GetHallConfiguration(ctx context.Context,
	in *cinema_service.GetHallConfigurationRequest) (*cinema_service.HallConfiguration, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaService.GetHallConfiguration")
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

func parsePeriods(startPeriod, endPeriod string) (start time.Time, end time.Time, err error) {
	start, err = time.Parse(time.RFC3339, startPeriod)
	if err != nil {
		err = fmt.Errorf("invalid start period value, it must be RFC3339 layout value: %s", startPeriod)
		return
	}
	end, err = time.Parse(time.RFC3339, endPeriod)
	if err != nil {
		err = fmt.Errorf("invalid start period value, it must be RFC3339 layout value: %s", endPeriod)
		return
	}

	return
}
