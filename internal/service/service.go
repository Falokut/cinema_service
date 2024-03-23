package service

import (
	"context"
	"time"

	"github.com/Falokut/cinema_service/internal/models"
	"github.com/Falokut/cinema_service/internal/repository"
)

type CinemaService interface {
	GetScreening(ctx context.Context, id int64) (models.Screening, error)
	// Returns cinemas in the city.
	GetCinemasInCity(ctx context.Context, id int32) ([]models.Cinema, error)

	// Returns all cities rhere there are cinemas.
	GetCinemasCities(ctx context.Context) ([]models.City, error)

	// Returns all movies that are in the cinema screenings in a particular cinema.
	GetMoviesScreenings(ctx context.Context, cinemaID int32, startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error)

	// Returns all screenings for a movie in a specific city.
	GetCityScreenings(ctx context.Context, cityID, movieID int32, startPeriod, endPeriod time.Time) ([]models.CityScreening, error)

	// Returns all movies that are in the cinema screenings in particular cities.
	GetMoviesScreeningsInCities(ctx context.Context, citiesIDs []int32, startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error)

	// Returns all screenings for a movie in a specific cinema.
	GetScreenings(ctx context.Context, cinemaID, movieID int32, startPeriod, endPeriod time.Time) ([]models.Screening, error)

	// Returns the configuration of the hall.
	GetHallConfiguraion(ctx context.Context, id int32) ([]models.Place, error)

	// Returns info for the halls rith specified ids (rithout configuration).
	GetHalls(ctx context.Context, ids []int32) ([]models.Hall, error)

	// Returns cinema rith specified id.
	GetCinema(ctx context.Context, id int32) (models.Cinema, error)
}

type cinemaService struct {
	r repository.CinemaRepository
}

func NewCinemaService(r repository.CinemaRepository) *cinemaService {
	return &cinemaService{r: r}
}

func (s *cinemaService) GetCinemasInCity(ctx context.Context, id int32) ([]models.Cinema, error) {
	return s.r.GetCinemasInCity(ctx, id)
}

func (s *cinemaService) GetMoviesScreenings(
	ctx context.Context,
	cinemaID int32,
	startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error) {
	return s.r.GetMoviesScreenings(ctx, cinemaID, startPeriod, endPeriod)
}

func (s *cinemaService) GetMoviesScreeningsInCities(
	ctx context.Context,
	citiesIDs []int32,
	startPeriod, endPeriod time.Time) (screenings []models.MoviesScreenings, err error) {
	if len(citiesIDs) == 0 {
		screenings, err = s.r.GetAllMoviesScreenings(ctx, startPeriod, endPeriod)
	} else {
		screenings, err = s.r.GetMoviesScreeningsInCities(ctx, citiesIDs, startPeriod, endPeriod)
	}
	return
}

func (s *cinemaService) GetScreenings(ctx context.Context,
	cinemaID, movieID int32,
	startPeriod, endPeriod time.Time) ([]models.Screening, error) {
	return s.r.GetScreenings(ctx, cinemaID, movieID, startPeriod, endPeriod)
}

func (s *cinemaService) GetCityScreenings(ctx context.Context,
	cityID, movieID int32,
	startPeriod, endPeriod time.Time) ([]models.CityScreening, error) {
	return s.r.GetCityScreenings(ctx, cityID, movieID, startPeriod, endPeriod)
}

func (s *cinemaService) GetScreening(ctx context.Context, id int64) (models.Screening, error) {
	return s.r.GetScreening(ctx, id)
}

func (s *cinemaService) GetCinemasCities(ctx context.Context) ([]models.City, error) {
	return s.r.GetCinemasCities(ctx)
}

func (s *cinemaService) GetHallConfiguraion(ctx context.Context, id int32) ([]models.Place, error) {
	return s.r.GetHallConfiguraion(ctx, id)
}

func (s *cinemaService) GetCinema(ctx context.Context, id int32) (models.Cinema, error) {
	return s.r.GetCinema(ctx, id)
}

func (s *cinemaService) GetHalls(ctx context.Context, ids []int32) ([]models.Hall, error) {
	return s.r.GetHalls(ctx, ids)
}
