package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func buildMySQLDSN(p mysqlProfile) string {
	params := p.Params
	if params == "" {
		params = "parseTime=true"
	}
	// DSN for driver
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s",
		p.User, p.Password, p.Host, p.Port, p.DBName, params)
}

func buildMySQLURL(p mysqlProfile) string {
	params := p.Params
	if params == "" {
		params = "parseTime=true"
	}
	// URL form for DATABASE_URL
	return fmt.Sprintf("mysql://%s:%s@%s:%s/%s?%s",
		p.User, p.Password, p.Host, p.Port, p.DBName, params)
}

func testMySQLConn(p mysqlProfile) error {
	dsn := buildMySQLDSN(p)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open failed: %w", err)
	}
	defer func() { _ = db.Close() }() // satisfy errcheck

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	fmt.Println("✅ MySQL connection successful")
	return nil
}
