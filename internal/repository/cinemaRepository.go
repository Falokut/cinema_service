package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

type cinemaRepository struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

func NewCinemaRepository(logger *logrus.Logger, db *sqlx.DB) *cinemaRepository {
	return &cinemaRepository{
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

func (r *cinemaRepository) GetCinemasInCity(ctx context.Context, id int32) ([]Cinema, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaRepository.GetCinemasInCity")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`SELECT id,name,address, ST_AsText(coordinates) AS coordinates
								FROM %s
								WHERE city_id=$1
								ORDER BY id`,
		cinemasTableName)

	var cinemas []Cinema
	err = r.db.SelectContext(ctx, &cinemas, query, id)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []Cinema{}, err
	}

	return cinemas, nil
}

func (r *cinemaRepository) GetCinemasCities(ctx context.Context) ([]City, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaRepository.GetCinemasCities")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf("SELECT * FROM %s WHERE id=ANY(SELECT DISTINCT city_id FROM %s) ORDER BY id",
		citiesTableName, cinemasTableName)

	var cities []City
	err = r.db.SelectContext(ctx, &cities, query)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []City{}, err
	}

	return cities, nil
}

type previewScreening struct {
	MovieID         int32  `json:"movie_id" db:"movie_id"`
	ScreeningsTypes string `json:"screenings_types" db:"screenings_types"`
	HallsTypes      string `json:"halls_types" db:"halls_types"`
}

func (r *cinemaRepository) GetPreviewScreenings(ctx context.Context,
	cinemaID int32, startPeriod, endPeriod time.Time) ([]PreviewScreening, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaRepository.GetPreviewScreenings")
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
		return []PreviewScreening{}, err
	}

	res := make([]PreviewScreening, len(previews))
	for i, screening := range previews {
		res[i] = PreviewScreening{
			MovieID:         screening.MovieID,
			HallsTypes:      convertSQLArray(screening.HallsTypes),
			ScreeningsTypes: convertSQLArray(screening.ScreeningsTypes),
		}
	}

	return res, nil
}

func (r *cinemaRepository) GetScreenings(ctx context.Context,
	cinemaID, movieID int32, startPeriod, endPeriod time.Time) ([]Screening, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaRepository.GetScreenings")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`
		SELECT %[1]s.id, movie_id, %[2]s.name AS screening_type, hall_id, ticket_price,start_time
		FROM %[1]s JOIN %[2]s ON screening_type_id=%[2]s.id 
		WHERE hall_id=ANY(SELECT id FROM %[3]s WHERE cinema_id=$1) AND movie_id=$2 AND start_time>=$3 AND start_time<=$4
		ORDER BY start_time`,
		screeningsTableName, screeningTypeTableName, hallsTableName)

	var screenings []Screening
	err = r.db.SelectContext(ctx, &screenings, query, cinemaID, movieID, startPeriod, endPeriod)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []Screening{}, err
	}

	return screenings, nil
}

func (r *cinemaRepository) GetHallConfiguraion(ctx context.Context, id int32) ([]Place, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "cinemaRepository.GetHallConfiguraion")
	defer span.Finish()
	var err error
	defer span.SetTag("error", err != nil)

	query := fmt.Sprintf(`SELECT row, seat, grid_pos_x, grid_pos_y
								FROM %s
								WHERE hall_id=$1
								ORDER BY row,seat`,
		hallsConfigurationsTableName)
	var places []Place
	err = r.db.SelectContext(ctx, &places, query, id)
	if err != nil {
		r.logger.Errorf("err: %v query: %s", err.Error(), query)
		return []Place{}, err
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
