package driver

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("Error opening mySql DB: %w", err)
	}
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("Error pinging mySql DB: %w", err)
	}
	return db, nil
}
