package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Полная конфигурация проекта
type Config struct {
	Database  DBConfig
	RabbitMQ  MQConfig
	WebSocket WSConfig
	Services  ServicesConfig
	JWT       JWTConfig
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string // DB_SSLMODE (disable по умолчанию)
}

type MQConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	VHost    string // RABBITMQ_VHOST ("/" по умолчанию)
}

type WSConfig struct {
	Port int
}

type ServicesConfig struct {
	RideServicePort           int
	DriverLocationServicePort int
	AdminServicePort          int
}

type JWTConfig struct {
	PassengerSecret string
	DriverSecret    string
	AdminSecret     string
}

// Load — чисто из ENV (простой, безопасный дефолт)
func Load() Config {
	return mergeConfig(nil)
}

// LoadFrom — YAML (если указан) + ENV перекрывает YAML
// Поддерживает простую YAML-структуру по ТЗ.
// Внутри не используется никаких внешних либ.
func LoadFrom(path string) (Config, error) {
	kv := map[string]map[string]string{} // section -> key -> value
	if path != "" {
		if err := parseSimpleYAML(path, kv); err != nil {
			return Config{}, err
		}
	}
	cfg := mergeConfig(kv)
	return cfg, nil
}

// mergeConfig: приоритет ENV > YAML > defaults
func mergeConfig(yaml map[string]map[string]string) Config {
	// Helpers to get values with precedence
	getStr := func(envKey, section, key, def string) string {
		if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
			return v
		}
		if yaml != nil {
			if sec, ok := yaml[section]; ok {
				if v, ok := sec[key]; ok && v != "" {
					return v
				}
			}
		}
		return def
	}
	getInt := func(envKey, section, key string, def int) int {
		if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
		if yaml != nil {
			if sec, ok := yaml[section]; ok {
				if v, ok := sec[key]; ok && v != "" {
					if n, err := strconv.Atoi(v); err == nil {
						return n
					}
				}
			}
		}
		return def
	}

	cfg := Config{}

	// Database
	cfg.Database.Host = getStr("DB_HOST", "database", "host", "localhost")
	cfg.Database.Port = getInt("DB_PORT", "database", "port", 5432)
	cfg.Database.User = getStr("DB_USER", "database", "user", "ridehail_user")
	cfg.Database.Password = getStr("DB_PASSWORD", "database", "password", "ridehail_pass")
	cfg.Database.Database = getStr("DB_NAME", "database", "database", "ridehail_db")
	cfg.Database.SSLMode = getStr("DB_SSLMODE", "database", "sslmode", "disable")

	// RabbitMQ
	cfg.RabbitMQ.Host = getStr("RABBITMQ_HOST", "rabbitmq", "host", "localhost")
	cfg.RabbitMQ.Port = getInt("RABBITMQ_PORT", "rabbitmq", "port", 5672)
	cfg.RabbitMQ.User = getStr("RABBITMQ_USER", "rabbitmq", "user", "guest")
	cfg.RabbitMQ.Password = getStr("RABBITMQ_PASSWORD", "rabbitmq", "password", "guest")
	cfg.RabbitMQ.VHost = getStr("RABBITMQ_VHOST", "rabbitmq", "vhost", "/")

	// WebSocket
	cfg.WebSocket.Port = getInt("WS_PORT", "websocket", "port", 8080)

	// Services
	cfg.Services.RideServicePort = getInt("RIDE_SERVICE_PORT", "services", "ride_service", 3000)
	cfg.Services.DriverLocationServicePort = getInt("DRIVER_LOCATION_SERVICE_PORT", "services", "driver_location_service", 3001)
	cfg.Services.AdminServicePort = getInt("ADMIN_SERVICE_PORT", "services", "admin_service", 3004)

	// JWT (три секрета по ролям)
	cfg.JWT.PassengerSecret = getStr("JWT_PASSENGER_SECRET", "jwt", "passenger_secret", "passenger_secret")
	cfg.JWT.DriverSecret = getStr("JWT_DRIVER_SECRET", "jwt", "driver_secret", "driver_secret")
	cfg.JWT.AdminSecret = getStr("JWT_ADMIN_SECRET", "jwt", "admin_secret", "admin_secret")

	return cfg
}

// parseSimpleYAML — очень простой парсер YAML вида:
// section:
//
//	key: value
//
// Поддерживает ${ENV:-default} в value.
// Не поддерживает вложенность глубже 1 уровня, массивы, кавычки и пр.
func parseSimpleYAML(path string, out map[string]map[string]string) error {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer f.Close()

	section := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// новая секция
		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") && !strings.Contains(line, "\t") {
			section = strings.TrimSuffix(line, ":")
			if out[section] == nil {
				out[section] = map[string]string{}
			}
			continue
		}
		// ключ: значение в секции
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// уберём возможные кавычки вокруг значения
		val = strings.Trim(val, `"'`)

		// поддержка ${VAR:-default}
		if strings.HasPrefix(val, "${") && strings.HasSuffix(val, "}") {
			val = expandEnv(val)
		}

		if section == "" {
			// ключ вне секции — игнорируем (или можно положить в секцию "root")
			continue
		}
		out[section][key] = val
	}
	return sc.Err()
}

func expandEnv(expr string) string {
	// expr вида ${VAR:-default} или ${VAR}
	inner := strings.TrimSuffix(strings.TrimPrefix(expr, "${"), "}")
	parts := strings.SplitN(inner, ":-", 2)
	env := os.Getenv(parts[0])
	if env != "" {
		return env
	}
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}
