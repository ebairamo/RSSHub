package storage

import (
	"database/sql"

	_ "github.com/lib/pq" // Драйвер PostgreSQL
)

// ConnectToDB устанавливает соединение с базой данных
func ConnectToDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
