package postgresrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Falokut/cinema_service/internal/models"
	"github.com/Falokut/cinema_service/internal/repository"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

type CinemaRepository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

func NewCinemaRepository(logger *logrus.Logger, db *sqlx.DB) *CinemaRepository {
	return &CinemaRepository{
		logger: logger,
		db:     db,
	}
}

const (
	cinemasTableName             = "cinemas"
	citiesTableName              = "cities"
	screeningTypeTableName       = "screenings_types"
	hallsTypesTableName          = "halls_types"
	hallsTableName               = "halls"
	screeningsTableName          = "screenings"
	hallsConfigurationsTableName = "halls_configurations"
)

func (r *CinemaRepository) GetCinemasInCity(ctx context.Context, id int32) ([]models.Cinema, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaRepository.GetCinemasInCity")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`SELECT id,name,address, ST_AsText(coordinates) AS coordinates
								FROM %s
								WHERE city_id=$1
								ORDER BY id`,
		cinemasTableName)

	var cinemas []models.Cinema
	err = r.db.SelectContext(ctx, &cinemas, query, id)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []models.Cinema{}, err
	}

	return cinemas, nil
}

func (r *CinemaRepository) GetCinemasCities(ctx context.Context) ([]models.City, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaRepository.GetCinemasCities")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf("SELECT * FROM %s WHERE id=ANY(SELECT DISTINCT city_id FROM %s) ORDER BY id",
		citiesTableName, cinemasTableName)

	var cities []models.City
	err = r.db.SelectContext(ctx, &cities, query)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []models.City{}, err
	}

	return cities, nil
}

type previewScreening struct {
	MovieId         int32  `db:"movie_id"`
	ScreeningsTypes string `db:"screenings_types"`
	HallsTypes      string `db:"halls_types"`
}

func (r *CinemaRepository) GetMoviesScreenings(ctx context.Context,
	cinemaID int32, startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaRepository.GetMoviesScreenings")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`SELECT movie_id, ARRAY_AGG(DISTINCT %[1]s.name) AS screenings_types,
		ARRAY_AGG(DISTINCT %[2]s.name) AS halls_types 
		FROM %[3]s JOIN %[1]s ON screening_type_id=%[1]s.id 
		JOIN %[4]s ON hall_id=%[4]s.id JOIN %[2]s ON hall_type_id=%[2]s.type_id 
		WHERE cinema_id=$1 AND start_time>=$2 AND start_time<=$3 
		GROUP BY movie_id`,
		screeningTypeTableName, hallsTypesTableName, screeningsTableName, hallsTableName)

	var previews []previewScreening
	err = r.db.SelectContext(ctx, &previews, query, cinemaID, startPeriod, endPeriod)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []models.MoviesScreenings{}, err
	}

	res := make([]models.MoviesScreenings, len(previews))
	for i, screening := range previews {
		res[i] = models.MoviesScreenings{
			MovieId:         screening.MovieId,
			HallsTypes:      convertSQLArray(screening.HallsTypes),
			ScreeningsTypes: convertSQLArray(screening.ScreeningsTypes),
		}
	}

	return res, nil
}

func (r *CinemaRepository) GetAllMoviesScreenings(ctx context.Context,
	startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx,
		"CinemaRepository.GetAllMoviesScreenings")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`SELECT movie_id, ARRAY_AGG(DISTINCT %[1]s.name) AS screenings_types,
		ARRAY_AGG(DISTINCT %[2]s.name) AS halls_types 
		FROM %[3]s JOIN %[1]s ON screening_type_id=%[1]s.id 
		JOIN %[4]s ON hall_id=%[4]s.id JOIN %[2]s ON hall_type_id=%[2]s.type_id 
		WHERE start_time>=$1 AND start_time<=$2 
		GROUP BY movie_id`,
		screeningTypeTableName, hallsTypesTableName, screeningsTableName, hallsTableName)

	var previews []previewScreening
	err = r.db.SelectContext(ctx, &previews, query, startPeriod, endPeriod)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []models.MoviesScreenings{}, err
	}

	res := make([]models.MoviesScreenings, len(previews))
	for i, screening := range previews {
		res[i] = models.MoviesScreenings{
			MovieId:         screening.MovieId,
			HallsTypes:      convertSQLArray(screening.HallsTypes),
			ScreeningsTypes: convertSQLArray(screening.ScreeningsTypes),
		}
	}

	return res, nil
}

func (r *CinemaRepository) GetMoviesScreeningsInCities(ctx context.Context,
	citiesIds []int32, startPeriod, endPeriod time.Time) ([]models.MoviesScreenings, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx,
		"CinemaRepository.GetMoviesScreeningsInCities")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`SELECT movie_id, ARRAY_AGG(DISTINCT %[1]s.name) AS screenings_types,
		ARRAY_AGG(DISTINCT %[2]s.name) AS halls_types 
		FROM %[3]s JOIN %[1]s ON screening_type_id=%[1]s.id 
		JOIN %[4]s ON hall_id=%[4]s.id JOIN %[2]s ON hall_type_id=%[2]s.type_id 
		WHERE cinema_id=ANY(SELECT id FROM %[5]s WHERE city_id=ANY($1)) AND start_time>=$2 AND start_time<=$3 
		GROUP BY movie_id`,
		screeningTypeTableName, hallsTypesTableName, screeningsTableName, hallsTableName, cinemasTableName)

	var previews []previewScreening
	err = r.db.SelectContext(ctx, &previews, query, citiesIds, startPeriod, endPeriod)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []models.MoviesScreenings{}, err
	}

	res := make([]models.MoviesScreenings, len(previews))
	for i, screening := range previews {
		res[i] = models.MoviesScreenings{
			MovieId:         screening.MovieId,
			HallsTypes:      convertSQLArray(screening.HallsTypes),
			ScreeningsTypes: convertSQLArray(screening.ScreeningsTypes),
		}
	}

	return res, nil
}

func (r *CinemaRepository) GetCityScreenings(ctx context.Context,
	cityId, movieId int32, startPeriod, endPeriod time.Time) ([]models.CityScreening, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaRepository.GetCityScreenings")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`
			SELECT %[1]s.id, %[2]s.name AS screening_type, hall_id, ticket_price,start_time, cinema_id 
			FROM %[1]s JOIN %[2]s ON screening_type_id=%[2]s.id 
			JOIN %[3]s ON hall_id = %[3]s.id 
			JOIN %[4]s ON cinema_id = %[4]s.id 
			WHERE city_id=$1 AND movie_id=$2 AND start_time>=$3 AND start_time<=$4 
			ORDER BY start_time;`,
		screeningsTableName, screeningTypeTableName, hallsTableName, cinemasTableName)

	var screenings []models.CityScreening
	err = r.db.SelectContext(ctx, &screenings, query, cityId, movieId, startPeriod, endPeriod)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []models.CityScreening{}, err
	}

	return screenings, nil
}
func (r *CinemaRepository) GetScreenings(ctx context.Context,
	cinemaID, movieID int32, startPeriod, endPeriod time.Time) ([]models.Screening, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaRepository.GetScreenings")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`
		SELECT %[1]s.id, movie_id, %[2]s.name AS screening_type, hall_id, ticket_price,start_time
		FROM %[1]s JOIN %[2]s ON screening_type_id=%[2]s.id 
		WHERE hall_id=ANY(SELECT id FROM %[3]s WHERE cinema_id=$1) AND movie_id=$2 AND start_time>=$3 AND start_time<=$4
		ORDER BY start_time;`,
		screeningsTableName, screeningTypeTableName, hallsTableName)

	var screenings []models.Screening
	err = r.db.SelectContext(ctx, &screenings, query, cinemaID, movieID, startPeriod, endPeriod)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []models.Screening{}, err
	}

	return screenings, nil
}

func (r *CinemaRepository) GetScreening(ctx context.Context, id int64) (models.Screening, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaRepository.GetScreening")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`
	SELECT  %[2]s.name AS screening_type,hall_id,ticket_price,start_time,cinema_id,movie_id 
	FROM %[1]s JOIN %[2]s ON screening_type_id=%[2]s.id 
	JOIN %[3]s ON hall_id = %[3]s.id 
	WHERE %[1]s.id=$1;`, screeningsTableName, screeningTypeTableName, hallsTableName)

	var screening models.Screening
	err = r.db.GetContext(ctx, &screening, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Screening{}, repository.ErrNotFound
	}
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return models.Screening{}, err
	}

	return screening, nil
}

func (r *CinemaRepository) GetCinema(ctx context.Context, id int32) (models.Cinema, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaRepository.GetCinema")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`SELECT id,name,address, ST_AsText(coordinates) AS coordinates
	FROM %s WHERE id=$1`, cinemasTableName)

	var Cinema models.Cinema
	err = r.db.GetContext(ctx, &Cinema, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Cinema{}, repository.ErrNotFound
	}

	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return models.Cinema{}, err
	}

	return Cinema, nil
}

func (r *CinemaRepository) GetHalls(ctx context.Context, ids []int32) ([]models.Hall, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaRepository.GetHalls")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`SELECT id, COALESCE(%[1]s.name,'') AS hall_type, %[2]s.name AS name, hall_size AS size
	FROM %[2]s LEFT JOIN %[1]s ON hall_type_id=type_id
	WHERE id=ANY($1)`, hallsTypesTableName, hallsTableName)
	var halls []models.Hall
	err = r.db.SelectContext(ctx, &halls, query, ids)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []models.Hall{}, err
	}
	return halls, nil
}

func (r *CinemaRepository) GetHallConfiguraion(ctx context.Context, id int32) ([]models.Place, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CinemaRepository.GetHallConfiguraion")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`SELECT row, seat, grid_pos_x, grid_pos_y
								FROM %s
								WHERE hall_id=$1
								ORDER BY row,seat`,
		hallsConfigurationsTableName)
	var places []models.Place
	err = r.db.SelectContext(ctx, &places, query, id)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []models.Place{}, err
	}

	return places, nil
}

func convertSQLArray(str string) []string {
	if strings.EqualFold(str, "{NULL}") {
		return []string{}
	}

	str = strings.Trim(str, "{}")
	return strings.Split(str, ",")
}