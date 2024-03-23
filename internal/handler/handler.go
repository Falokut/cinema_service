package handler

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Falokut/cinema_service/internal/models"
	"github.com/Falokut/cinema_service/internal/service"
	cinema_service "github.com/Falokut/cinema_service/pkg/cinema_service/v1/protos"
	"github.com/mennanov/fmutils"
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

	GetScreening(ctx context.Context, id int64) (*cinema_service.GetScreeningResponse, error)
}

type CinemaServiceHandler struct {
	cinema_service.UnimplementedCinemaServiceV1Server
	s service.CinemaService
}

func NewCinemaServiceHandler(s service.CinemaService) *CinemaServiceHandler {
	return &CinemaServiceHandler{s: s}
}

func (h *CinemaServiceHandler) GetCinemasInCity(ctx context.Context,
	in *cinema_service.GetCinemasInCityRequest) (cinemas *cinema_service.Cinemas, err error) {
	defer h.handleError(&err)

	modelsCinema, err := h.s.GetCinemasInCity(ctx, in.CityID)
	if err != nil {
		return
	}

	cinemas = &cinema_service.Cinemas{
		Cinemas: make([]*cinema_service.Cinema, len(modelsCinema)),
	}

	for i := range modelsCinema {
		cinemas.Cinemas[i] = cinemaFromModels(&modelsCinema[i])
	}

	return
}

func (h *CinemaServiceHandler) GetMoviesScreenings(ctx context.Context,
	in *cinema_service.GetMoviesScreeningsRequest) (screenings *cinema_service.PreviewScreenings, err error) {
	defer h.handleError(&err)
	start, end, err := parsePeriods(in.StartPeriod, in.EndPeriod)
	if err != nil {
		return
	}
	modelsScreenings, err := h.s.GetMoviesScreenings(ctx, in.CinemaID, start, end)
	if err != nil {
		return
	}
	screenings = previewScreeningsFromModel(modelsScreenings)
	return
}

func (h *CinemaServiceHandler) GetMoviesScreeningsInCities(ctx context.Context,
	in *cinema_service.GetMoviesScreeningsInCitiesRequest) (screenings *cinema_service.PreviewScreenings, err error) {
	defer h.handleError(&err)

	start, end, err := parsePeriods(in.StartPeriod, in.EndPeriod)
	if err != nil {
		return
	}

	var ids []int32
	if in.GetCitiesIds() != "" {
		citiesIDs := strings.ReplaceAll(in.GetCitiesIds(), `"`, "")
		err = checkIds(citiesIDs)
		if err != nil {
			return
		}
		ids = convertStringsSlice(strings.Split(citiesIDs, ","))
	}

	modelsScreenings, err := h.s.GetMoviesScreeningsInCities(ctx, ids, start, end)
	if err != nil {
		return
	}
	screenings = previewScreeningsFromModel(modelsScreenings)
	return
}

func previewScreeningsFromModel(screenings []models.MoviesScreenings) *cinema_service.PreviewScreenings {
	converted := &cinema_service.PreviewScreenings{
		Screenings: make([]*cinema_service.PreviewScreening, len(screenings)),
	}

	for i := range screenings {
		converted.Screenings[i] = &cinema_service.PreviewScreening{
			MovieID:         screenings[i].MovieID,
			ScreeningsTypes: screenings[i].ScreeningsTypes,
			HallsTypes:      screenings[i].HallsTypes,
		}
	}

	return converted
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

func hallConfigurationFromModel(places []models.Place) *cinema_service.HallConfiguration {
	converted := &cinema_service.HallConfiguration{
		Place: make([]*cinema_service.Place, len(places)),
	}

	for i := range places {
		converted.Place[i] = &cinema_service.Place{
			Row:      places[i].Row,
			Seat:     places[i].Seat,
			GridPosX: places[i].GridPosX,
			GridPosY: places[i].GridPosY,
		}
	}

	return converted
}

func (h *CinemaServiceHandler) GetScreenings(ctx context.Context,
	in *cinema_service.GetScreeningsRequest) (screenings *cinema_service.Screenings, err error) {
	defer h.handleError(&err)

	start, end, err := parsePeriods(in.StartPeriod, in.EndPeriod)
	if err != nil {
		return
	}

	modelsScreenings, err := h.s.GetScreenings(ctx, in.CinemaID, in.MovieID, start, end)
	if err != nil {
		return
	}

	screenings = &cinema_service.Screenings{
		Screenings: make([]*cinema_service.Screening, len(modelsScreenings)),
	}

	for i := range modelsScreenings {
		screenings.Screenings[i] = &cinema_service.Screening{
			ScreeningID:   modelsScreenings[i].ScreeningID,
			MovieID:       modelsScreenings[i].MovieID,
			ScreeningType: modelsScreenings[i].ScreeningType,
			StartTime:     &cinema_service.Timestamp{FormattedTimestamp: modelsScreenings[i].StartTime.Format(time.RFC3339)},
			HallID:        modelsScreenings[i].HallID,
			TicketPrice:   priceFromString(modelsScreenings[i].TicketPrice),
		}
	}

	return
}

func (h *CinemaServiceHandler) GetScreeningsInCity(ctx context.Context,
	in *cinema_service.GetScreeningsInCityRequest) (screenings *cinema_service.CityScreenings, err error) {
	defer h.handleError(&err)

	start, end, err := parsePeriods(in.StartPeriod, in.EndPeriod)
	if err != nil {
		return
	}

	modelsScreenings, err := h.s.GetCityScreenings(ctx, in.CityID, in.MovieID, start, end)
	if err != nil {
		return
	}

	screenings = &cinema_service.CityScreenings{
		Screenings: make([]*cinema_service.CityScreening, len(modelsScreenings)),
	}
	for i := range modelsScreenings {
		screenings.Screenings[i] = &cinema_service.CityScreening{
			ScreeningID:   modelsScreenings[i].ScreeningID,
			CinemaID:      modelsScreenings[i].CinemaID,
			ScreeningType: modelsScreenings[i].ScreeningType,
			StartTime:     formattedTimestampFromTime(modelsScreenings[i].StartTime),
			HallID:        modelsScreenings[i].HallID,
			TicketPrice:   priceFromString(modelsScreenings[i].TicketPrice),
		}
	}

	return
}

func formattedTimestampFromTime(t time.Time) *cinema_service.Timestamp {
	return &cinema_service.Timestamp{FormattedTimestamp: t.Format(time.RFC3339)}
}

func (h *CinemaServiceHandler) GetScreening(ctx context.Context,
	in *cinema_service.GetScreeningRequest) (screening *cinema_service.GetScreeningResponse, err error) {
	defer h.handleError(&err)

	if in.Mask != nil && !in.Mask.IsValid(&cinema_service.GetScreeningResponse{}) {
		return nil, status.Error(codes.InvalidArgument, "invalid mask value")
	}

	modelsScreening, err := h.s.GetScreening(ctx, in.ScreeningID)
	if err != nil {
		return
	}

	configuration := &cinema_service.HallConfiguration{}
	var needConfiguration = in.Mask == nil || len(in.Mask.Paths) == 0
	if in.Mask != nil {
		for i := range in.Mask.Paths {
			if in.Mask.Paths[i] == "hall_configuration" {
				needConfiguration = true
				break
			}
		}
	}

	if needConfiguration {
		configuration, err = h.GetHallConfiguration(ctx,
			&cinema_service.GetHallConfigurationRequest{
				HallID: modelsScreening.HallID,
			})
		if err != nil {
			return
		}
	}

	screening = &cinema_service.GetScreeningResponse{
		CinemaID:          modelsScreening.CinemaID,
		MovieID:           modelsScreening.MovieID,
		ScreeningType:     modelsScreening.ScreeningType,
		StartTime:         formattedTimestampFromTime(modelsScreening.StartTime),
		HallID:            modelsScreening.HallID,
		TicketPrice:       priceFromString(modelsScreening.TicketPrice),
		HallConfiguration: configuration,
	}

	if in.Mask != nil {
		fmutils.Filter(screening, in.Mask.Paths)
	}

	return
}

func (h *CinemaServiceHandler) GetCinemasCities(ctx context.Context,
	_ *emptypb.Empty) (cities *cinema_service.Cities, err error) {
	defer h.handleError(&err)

	modelsCities, err := h.s.GetCinemasCities(ctx)
	if err != nil {
		return
	}

	cities = &cinema_service.Cities{Cities: make([]*cinema_service.City, len(modelsCities))}
	for i := range modelsCities {
		cities.Cities[i] = &cinema_service.City{
			CityID: modelsCities[i].ID,
			Name:   modelsCities[i].Name,
		}
	}

	return
}

func (h *CinemaServiceHandler) GetHallConfiguration(ctx context.Context,
	in *cinema_service.GetHallConfigurationRequest) (configuration *cinema_service.HallConfiguration, err error) {
	defer h.handleError(&err)

	places, err := h.s.GetHallConfiguraion(ctx, in.HallID)
	if err != nil {
		return
	}

	configuration = hallConfigurationFromModel(places)
	return
}

func (h *CinemaServiceHandler) GetCinema(ctx context.Context,
	in *cinema_service.GetCinemaRequest) (cinema *cinema_service.Cinema, err error) {
	defer h.handleError(&err)

	modelsCinema, err := h.s.GetCinema(ctx, in.CinemaID)
	if err != nil {
		return
	}

	cinema = cinemaFromModels(&modelsCinema)
	return
}

func cinemaFromModels(cinema *models.Cinema) *cinema_service.Cinema {
	return &cinema_service.Cinema{
		CinemaID: cinema.ID,
		Name:     cinema.Name,
		Address:  cinema.Address,
		Coordinates: &cinema_service.Coordinates{
			Longitude: cinema.Coordinates.Longitude,
			Latityde:  cinema.Coordinates.Latityde,
		},
	}
}

func (h *CinemaServiceHandler) GetHalls(ctx context.Context,
	in *cinema_service.GetHallsRequest) (halls *cinema_service.Halls, err error) {
	defer h.handleError(&err)

	in.HallsIds = strings.ReplaceAll(in.HallsIds, `"`, "")
	if err = checkIds(in.HallsIds); err != nil {
		return
	}

	ids := convertStringsSlice(strings.Split(in.HallsIds, ","))
	modelsHalls, err := h.s.GetHalls(ctx, ids)
	if err != nil {
		return
	}
	halls = &cinema_service.Halls{
		Halls: make([]*cinema_service.Hall, len(modelsHalls)),
	}

	for i := range modelsHalls {
		halls.Halls[i] = &cinema_service.Hall{
			HallID:   modelsHalls[i].ID,
			HallSize: modelsHalls[i].Size,
			Name:     modelsHalls[i].Name,
			Type:     modelsHalls[i].Type,
		}
	}

	return
}

func parsePeriods(startPeriod, endPeriod *cinema_service.Timestamp) (start, end time.Time, err error) {
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
	if !regexp.MustCompile(`^\d+(,\d+)*$`).MatchString(val) {
		return status.Error(codes.InvalidArgument, "ids must contain only digits and commas")
	}

	return nil
}

func priceFromString(n string) *cinema_service.Price {
	nums := strings.Split(n, ".")
	if len(nums) != 2 {
		return nil
	}
	units, _ := strconv.ParseInt(nums[0], 10, 64)
	nanos, _ := strconv.ParseInt(nums[1], 10, 32)
	return &cinema_service.Price{Value: int32(units)*100 + int32(nanos)}
}

func (h *CinemaServiceHandler) handleError(err *error) {
	if err == nil || *err == nil {
		return
	}

	serviceErr := &models.ServiceError{}
	if errors.As(*err, &serviceErr) {
		*err = status.Error(convertServiceErrCodeToGrpc(serviceErr.Code), serviceErr.Msg)
	} else if _, ok := status.FromError(*err); !ok {
		e := *err
		*err = status.Error(codes.Unknown, e.Error())
	}
}

func convertServiceErrCodeToGrpc(code models.ErrorCode) codes.Code {
	switch code {
	case models.Internal:
		return codes.Internal
	case models.InvalidArgument:
		return codes.InvalidArgument
	case models.Unauthenticated:
		return codes.Unauthenticated
	case models.Conflict:
		return codes.AlreadyExists
	case models.NotFound:
		return codes.NotFound
	case models.Canceled:
		return codes.Canceled
	case models.DeadlineExceeded:
		return codes.DeadlineExceeded
	case models.PermissionDenied:
		return codes.PermissionDenied
	default:
		return codes.Unknown
	}
}
