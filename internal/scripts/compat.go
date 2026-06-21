package scripts

import "strings"

func hasEnvKey(keys map[string]string, names ...string) bool {
	for _, n := range names {
		if keys[n] != "" {
			return true
		}
	}
	return false
}

func envConn(keys map[string]string) string {
	return strings.ToLower(strings.TrimSpace(keys["DB_CONNECTION"]))
}

func scriptHasRegistryKey(s ScriptInfo, keys map[string]string) bool {
	for _, ek := range s.EnvKeys {
		if keys[ek] != "" {
			return true
		}
	}
	return false
}

func scriptCompatible(s ScriptInfo, keys map[string]string) bool {
	if !scriptHasRegistryKey(s, keys) {
		return false
	}
	switch s.ID {
	case "chk-mysql":
		return mysqlCompatible(keys)
	case "chk-postgres":
		return postgresCompatible(keys)
	case "chk-mongodb":
		return mongodbCompatible(keys)
	case "chk-redis":
		return redisCompatible(keys)
	default:
		return true
	}
}

func mysqlCompatible(keys map[string]string) bool {
	conn := envConn(keys)
	if conn == "pgsql" || conn == "postgres" || conn == "postgresql" || conn == "mongodb" || conn == "mongo" {
		return false
	}
	url := keys["DATABASE_URL"]
	if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
		return false
	}
	if strings.HasPrefix(url, "mysql://") {
		return true
	}
	switch conn {
	case "mysql", "mysqli", "mariadb":
		return hasEnvKey(keys, "DB_HOST", "MYSQL_HOST", "DATABASE_HOST", "DB_USERNAME", "MYSQL_USER", "DB_PASSWORD")
	case "":
		if hasEnvKey(keys, "MYSQL_HOST") {
			return true
		}
		return hasEnvKey(keys, "DB_HOST", "DB_USERNAME") && !hasEnvKey(keys, "PGHOST", "POSTGRES_HOST")
	default:
		return false
	}
}

func postgresCompatible(keys map[string]string) bool {
	conn := envConn(keys)
	if conn == "mysql" || conn == "mysqli" || conn == "mariadb" || conn == "mongodb" || conn == "mongo" || conn == "sqlsrv" {
		return false
	}
	url := keys["DATABASE_URL"]
	if strings.HasPrefix(url, "mysql://") {
		return false
	}
	if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
		return true
	}
	if hasEnvKey(keys, "POSTGRES_URL", "PGDATABASE_URL") {
		return true
	}
	switch conn {
	case "pgsql", "postgres", "postgresql":
		return hasEnvKey(keys, "DB_HOST", "PGHOST", "POSTGRES_HOST", "DATABASE_URL", "DB_USERNAME", "PGUSER")
	case "":
		return hasEnvKey(keys, "PGHOST", "POSTGRES_HOST", "PGUSER", "POSTGRES_USER")
	default:
		return false
	}
}

func mongodbCompatible(keys map[string]string) bool {
	conn := envConn(keys)
	if conn == "mysql" || conn == "mysqli" || conn == "mariadb" || conn == "pgsql" || conn == "postgres" || conn == "postgresql" {
		return false
	}
	if strings.HasPrefix(keys["MONGODB_URI"], "mysql://") {
		return false
	}
	if hasEnvKey(keys, "MONGODB_URI", "MONGO_URI", "MONGO_URL") {
		return true
	}
	return conn == "mongodb" || conn == "mongo"
}

func redisCompatible(keys map[string]string) bool {
	if !hasEnvKey(keys, "REDIS_HOST", "REDIS_URL") {
		return false
	}
	cache := strings.ToLower(strings.TrimSpace(keys["CACHE_DRIVER"] + keys["CACHE_STORE"]))
	switch cache {
	case "file", "database", "array", "apc", "cookie":
		return false
	}
	return true
}
