package db

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func testRedisConn(p redisProfile) error {
	addr := fmt.Sprintf("%s:%s", p.Host, p.Port)
	db := 0
	if p.DB != "" {
		var err error
		db, err = strconv.Atoi(p.DB)
		if err != nil {
			return fmt.Errorf("invalid db number %q: %w", p.DB, err)
		}
	}

	opts := &redis.Options{
		Addr:     addr,
		Password: p.Password,
		DB:       db,
	}
	if p.TLS {
		opts.TLSConfig = &tls.Config{} // use default TLS
	}

	client := redis.NewClient(opts)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	fmt.Println("✅ Redis connection successful")
	return nil
}
