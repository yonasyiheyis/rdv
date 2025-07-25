package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func testPgConn(p pgProfile) error {
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, p.DBName)
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	defer func() {
		_ = conn.Close(context.Background())
	}()

	if err = conn.Ping(context.Background()); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	fmt.Println("✅ Postgres connection successful")
	return nil
}
