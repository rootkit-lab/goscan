package scripts

import "testing"

func TestPostgresSkipsMySQL(t *testing.T) {
	keys := map[string]string{
		"DB_CONNECTION": "mysql",
		"DB_HOST":       "127.0.0.1",
		"DB_USERNAME":   "root",
		"DB_PASSWORD":   "secret",
	}
	if postgresCompatible(keys) {
		t.Fatal("postgres should not match mysql site")
	}
	if !mysqlCompatible(keys) {
		t.Fatal("mysql should match mysql site")
	}
}

func TestPostgresMatchesPgsql(t *testing.T) {
	keys := map[string]string{
		"DB_CONNECTION": "pgsql",
		"DB_HOST":       "127.0.0.1",
		"DB_USERNAME":   "postgres",
		"DB_PASSWORD":   "secret",
	}
	if !postgresCompatible(keys) {
		t.Fatal("postgres should match pgsql site")
	}
	if mysqlCompatible(keys) {
		t.Fatal("mysql should not match pgsql site")
	}
}

func TestPostgresURL(t *testing.T) {
	keys := map[string]string{
		"DATABASE_URL": "postgresql://user:pass@db.example.com:5432/app",
	}
	if !postgresCompatible(keys) {
		t.Fatal("expected postgres url match")
	}
}

func TestAmbiguousDBHostSkippedForPostgres(t *testing.T) {
	keys := map[string]string{
		"DB_HOST":     "127.0.0.1",
		"DB_USERNAME": "root",
	}
	if postgresCompatible(keys) {
		t.Fatal("ambiguous laravel mysql keys should not match postgres")
	}
}

func TestRedisSkipsFileCache(t *testing.T) {
	keys := map[string]string{
		"REDIS_HOST":    "127.0.0.1",
		"CACHE_DRIVER":  "file",
	}
	if redisCompatible(keys) {
		t.Fatal("redis checker should skip file cache sites")
	}
	if !redisCompatible(map[string]string{"REDIS_HOST": "127.0.0.1", "CACHE_DRIVER": "redis"}) {
		t.Fatal("redis cache should match")
	}
}
