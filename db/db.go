package db

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func ConnectToPostgresDb(dbConnStr string) (*sqlx.DB, error) {

	db, err := sqlx.Open("postgres", dbConnStr)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
