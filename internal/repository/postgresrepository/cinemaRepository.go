package postgresrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Falokut/cinema_service/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
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

func (r *CinemaRepository) GetCinemasInCity(ctx context.Context, id int32) (cinemas []models.Cinema, err error) {
	defer r.handleError(ctx, &err, "GetCinemasInCity")

	query := fmt.Sprintf(`SELECT id,name,address, ST_AsText(coordinates) AS coordinates
								FROM %s
								WHERE city_id=$1
								ORDER BY id`,
		cinemasTableName)

	err = r.db.SelectContext(ctx, &cinemas, query, id)
	return
}

func (r *CinemaRepository) GetCinemasCities(ctx context.Context) (cities []models.City, err error) {
	defer r.handleError(ctx, &err, "GetCinemasCities")

	query := fmt.Sprintf("SELECT * FROM %s WHERE id=ANY(SELECT DISTINCT city_id FROM %s) ORDER BY id",
		citiesTableName, cinemasTableName)

	err = r.db.SelectContext(ctx, &cities, query)
	return
}

type previewScreening struct {
	MovieID         int32  `db:"movie_id"`
	ScreeningsTypes string `db:"screenings_types"`
	HallsTypes      string `db:"halls_types"`
}

func (r *CinemaRepository) GetMoviesScreenings(ctx context.Context,
	cinemaID int32, startPeriod, endPeriod time.Time) (screenings []models.MoviesScreenings, err error) {
	defer r.handleError(ctx, &err, "GetMoviesScreenings")

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
		return
	}

	screenings = make([]models.MoviesScreenings, len(previews))
	for i, screening := range previews {
		screenings[i] = models.MoviesScreenings{
			MovieID:         screening.MovieID,
			HallsTypes:      convertSQLArray(screening.HallsTypes),
			ScreeningsTypes: convertSQLArray(screening.ScreeningsTypes),
		}
	}

	return
}

func (r *CinemaRepository) GetAllMoviesScreenings(ctx context.Context,
	startPeriod, endPeriod time.Time) (screenings []models.MoviesScreenings, err error) {
	defer r.handleError(ctx, &err, "GetMoviesScreeningsInCities")

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
		return
	}

	screenings = make([]models.MoviesScreenings, len(previews))
	for i, screening := range previews {
		screenings[i] = models.MoviesScreenings{
			MovieID:         screening.MovieID,
			HallsTypes:      convertSQLArray(screening.HallsTypes),
			ScreeningsTypes: convertSQLArray(screening.ScreeningsTypes),
		}
	}

	return
}

func (r *CinemaRepository) GetMoviesScreeningsInCities(ctx context.Context,
	citiesIDs []int32, startPeriod, endPeriod time.Time) (screenings []models.MoviesScreenings, err error) {
	defer r.handleError(ctx, &err, "GetMoviesScreeningsInCities")

	query := fmt.Sprintf(`SELECT movie_id, ARRAY_AGG(DISTINCT %[1]s.name) AS screenings_types,
		ARRAY_AGG(DISTINCT %[2]s.name) AS halls_types 
		FROM %[3]s JOIN %[1]s ON screening_type_id=%[1]s.id 
		JOIN %[4]s ON hall_id=%[4]s.id JOIN %[2]s ON hall_type_id=%[2]s.type_id 
		WHERE cinema_id=ANY(SELECT id FROM %[5]s WHERE city_id=ANY($1)) AND start_time>=$2 AND start_time<=$3 
		GROUP BY movie_id`,
		screeningTypeTableName, hallsTypesTableName, screeningsTableName, hallsTableName, cinemasTableName)

	var previews []previewScreening
	err = r.db.SelectContext(ctx, &previews, query, citiesIDs, startPeriod, endPeriod)
	if err != nil {
		return
	}

	screenings = make([]models.MoviesScreenings, len(previews))
	for i, screening := range previews {
		screenings[i] = models.MoviesScreenings{
			MovieID:         screening.MovieID,
			HallsTypes:      convertSQLArray(screening.HallsTypes),
			ScreeningsTypes: convertSQLArray(screening.ScreeningsTypes),
		}
	}

	return
}

func (r *CinemaRepository) GetCityScreenings(ctx context.Context,
	cityID, movieID int32, startPeriod, endPeriod time.Time) (screenings []models.CityScreening, err error) {
	defer r.handleError(ctx, &err, "GetCityScreenings")

	query := fmt.Sprintf(`
			SELECT %[1]s.id, %[2]s.name AS screening_type, hall_id, ticket_price,start_time, cinema_id 
			FROM %[1]s JOIN %[2]s ON screening_type_id=%[2]s.id 
			JOIN %[3]s ON hall_id = %[3]s.id 
			JOIN %[4]s ON cinema_id = %[4]s.id 
			WHERE city_id=$1 AND movie_id=$2 AND start_time>=$3 AND start_time<=$4 
			ORDER BY start_time;`,
		screeningsTableName, screeningTypeTableName, hallsTableName, cinemasTableName)

	err = r.db.SelectContext(ctx, &screenings, query, cityID, movieID, startPeriod, endPeriod)
	return
}

func (r *CinemaRepository) GetScreenings(ctx context.Context,
	cinemaID, movieID int32, startPeriod, endPeriod time.Time) (screenings []models.Screening, err error) {
	defer r.handleError(ctx, &err, "GetScreenings")

	query := fmt.Sprintf(`
		SELECT %[1]s.id, movie_id, %[2]s.name AS screening_type, hall_id, ticket_price,start_time
		FROM %[1]s JOIN %[2]s ON screening_type_id=%[2]s.id 
		WHERE hall_id=ANY(SELECT id FROM %[3]s WHERE cinema_id=$1) AND movie_id=$2 AND start_time>=$3 AND start_time<=$4
		ORDER BY start_time;`,
		screeningsTableName, screeningTypeTableName, hallsTableName)

	err = r.db.SelectContext(ctx, &screenings, query, cinemaID, movieID, startPeriod, endPeriod)
	return
}

func (r *CinemaRepository) GetScreening(ctx context.Context, id int64) (screening models.Screening, err error) {
	defer r.handleError(ctx, &err, "GetScreening")
	query := fmt.Sprintf(`
	SELECT  %[2]s.name AS screening_type,hall_id,ticket_price,start_time,cinema_id,movie_id 
	FROM %[1]s JOIN %[2]s ON screening_type_id=%[2]s.id 
	JOIN %[3]s ON hall_id = %[3]s.id 
	WHERE %[1]s.id=$1;`, screeningsTableName, screeningTypeTableName, hallsTableName)

	err = r.db.GetContext(ctx, &screening, query, id)
	return
}

func (r *CinemaRepository) GetCinema(ctx context.Context, id int32) (cinema models.Cinema, err error) {
	defer r.handleError(ctx, &err, "GetCinema")

	query := fmt.Sprintf(`SELECT id,name,address, ST_AsText(coordinates) AS coordinates
	FROM %s WHERE id=$1`, cinemasTableName)

	err = r.db.GetContext(ctx, &cinema, query, id)
	return
}

func (r *CinemaRepository) GetHalls(ctx context.Context, ids []int32) (halls []models.Hall, err error) {
	defer r.handleError(ctx, &err, "GetHalls")

	query := fmt.Sprintf(`SELECT id, COALESCE(%[1]s.name,'') AS hall_type, %[2]s.name AS name, hall_size AS size
	FROM %[2]s LEFT JOIN %[1]s ON hall_type_id=type_id
	WHERE id=ANY($1)`, hallsTypesTableName, hallsTableName)
	err = r.db.SelectContext(ctx, &halls, query, ids)
	return
}

func (r *CinemaRepository) GetHallConfiguraion(ctx context.Context, id int32) (places []models.Place, err error) {
	defer r.handleError(ctx, &err, "GetHallConfiguraion")

	query := fmt.Sprintf(`SELECT row, seat, grid_pos_x, grid_pos_y
								FROM %s
								WHERE hall_id=$1
								ORDER BY row,seat`,
		hallsConfigurationsTableName)
	err = r.db.SelectContext(ctx, &places, query, id)
	return
}

func convertSQLArray(str string) []string {
	if strings.EqualFold(str, "{NULL}") {
		return []string{}
	}

	str = strings.Trim(str, "{}")
	return strings.Split(str, ",")
}

func (r *CinemaRepository) handleError(ctx context.Context, err *error, functionName string) {
	if ctx.Err() != nil {
		var code models.ErrorCode
		switch {
		case errors.Is(ctx.Err(), context.Canceled):
			code = models.Canceled
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			code = models.DeadlineExceeded
		}
		*err = models.Error(code, ctx.Err().Error())
		r.logError(*err, functionName)
		return
	}

	if err == nil || *err == nil {
		return
	}

	r.logError(*err, functionName)
	var repoErr = &models.ServiceError{}
	if !errors.As(*err, &repoErr) {
		switch {
		case errors.Is(*err, sql.ErrNoRows):
			*err = models.Error(models.NotFound, "")
		case *err != nil:
			*err = models.Error(models.Internal, "repository internal error")
		}
	}
}

func (r *CinemaRepository) logError(err error, functionName string) {
	if err == nil {
		return
	}

	var repoErr = &models.ServiceError{}
	if errors.As(err, &repoErr) {
		r.logger.WithFields(
			logrus.Fields{
				"error.function.name": functionName,
				"error.msg":           repoErr.Msg,
				"error.code":          repoErr.Code,
			},
		).Error("cinema repository error occurred")
	} else {
		r.logger.WithFields(
			logrus.Fields{
				"error.function.name": functionName,
				"error.msg":           err.Error(),
			},
		).Error("cinema repository error occurred")
	}
}
