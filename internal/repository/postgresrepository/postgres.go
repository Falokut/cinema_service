package postgresrepository

import (
	"fmt"

	"github.com/Falokut/cinema_service/internal/repository"
	"github.com/jmoiron/sqlx"
)

func NewPostgreDB(cfg repository.DBConfig) (*sqlx.DB, error) {
	conStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBName, cfg.SSLMode)
	db, err := sqlx.Connect("pgx", conStr)

	if err != nil {
		return nil, err
	}

	return db, nil
}
