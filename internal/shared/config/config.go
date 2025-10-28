package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config — полная конфигурация проекта
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
	SSLMode  string
}

type MQConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	VHost    string
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
	Secret        string
	ExpiryMinutes int
}

// Load — загрузка из CONFIG_DIR (по умолчанию ./config) + ENV перекрывает
func Load() Config {
	configDir := getEnv("CONFIG_DIR", "./config")
	cfg := Config{}

	// Загружаем db.yaml
	dbPath := filepath.Join(configDir, "db.yaml")
	if dbKV, err := parseYAML(dbPath); err == nil {
		cfg.Database.Host = getStrWithEnv("DB_HOST", dbKV, "host", "localhost")
		cfg.Database.Port = getIntWithEnv("DB_PORT", dbKV, "port", 5432)
		cfg.Database.User = getStrWithEnv("DB_USER", dbKV, "user", "ridehail_user")
		cfg.Database.Password = getStrWithEnv("DB_PASSWORD", dbKV, "password", "ridehail_pass")
		cfg.Database.Database = getStrWithEnv("DB_NAME", dbKV, "database", "ridehail_db")
		cfg.Database.SSLMode = getStrWithEnv("DB_SSLMODE", dbKV, "sslmode", "disable")
	} else {
		// fallback to defaults + env
		cfg.Database.Host = getEnv("DB_HOST", "localhost")
		cfg.Database.Port = getEnvInt("DB_PORT", 5432)
		cfg.Database.User = getEnv("DB_USER", "ridehail_user")
		cfg.Database.Password = getEnv("DB_PASSWORD", "ridehail_pass")
		cfg.Database.Database = getEnv("DB_NAME", "ridehail_db")
		cfg.Database.SSLMode = getEnv("DB_SSLMODE", "disable")
	}

	// mq.yaml
	mqPath := filepath.Join(configDir, "mq.yaml")
	if mqKV, err := parseYAML(mqPath); err == nil {
		cfg.RabbitMQ.Host = getStrWithEnv("RABBITMQ_HOST", mqKV, "host", "localhost")
		cfg.RabbitMQ.Port = getIntWithEnv("RABBITMQ_PORT", mqKV, "port", 5672)
		cfg.RabbitMQ.User = getStrWithEnv("RABBITMQ_USER", mqKV, "user", "guest")
		cfg.RabbitMQ.Password = getStrWithEnv("RABBITMQ_PASSWORD", mqKV, "password", "guest")
		cfg.RabbitMQ.VHost = getStrWithEnv("RABBITMQ_VHOST", mqKV, "vhost", "/")
	} else {
		cfg.RabbitMQ.Host = getEnv("RABBITMQ_HOST", "localhost")
		cfg.RabbitMQ.Port = getEnvInt("RABBITMQ_PORT", 5672)
		cfg.RabbitMQ.User = getEnv("RABBITMQ_USER", "guest")
		cfg.RabbitMQ.Password = getEnv("RABBITMQ_PASSWORD", "guest")
		cfg.RabbitMQ.VHost = getEnv("RABBITMQ_VHOST", "/")
	}

	// ws.yaml
	wsPath := filepath.Join(configDir, "ws.yaml")
	if wsKV, err := parseYAML(wsPath); err == nil {
		cfg.WebSocket.Port = getIntWithEnv("WS_PORT", wsKV, "port", 8080)
	} else {
		cfg.WebSocket.Port = getEnvInt("WS_PORT", 8080)
	}

	// service.yaml
	svcPath := filepath.Join(configDir, "service.yaml")
	if svcKV, err := parseYAML(svcPath); err == nil {
		cfg.Services.RideServicePort = getIntWithEnv("RIDE_SERVICE_PORT", svcKV, "ride_service", 3000)
		cfg.Services.DriverLocationServicePort = getIntWithEnv("DRIVER_LOCATION_SERVICE_PORT", svcKV, "driver_location_service", 3001)
		cfg.Services.AdminServicePort = getIntWithEnv("ADMIN_SERVICE_PORT", svcKV, "admin_service", 3004)
	} else {
		cfg.Services.RideServicePort = getEnvInt("RIDE_SERVICE_PORT", 3000)
		cfg.Services.DriverLocationServicePort = getEnvInt("DRIVER_LOCATION_SERVICE_PORT", 3001)
		cfg.Services.AdminServicePort = getEnvInt("ADMIN_SERVICE_PORT", 3004)
	}

	// jwt.yaml
	jwtPath := filepath.Join(configDir, "jwt.yaml")
	if jwtKV, err := parseYAML(jwtPath); err == nil {
		// пробуем сначала секцию jwt.secret и jwt.expiry_minutes
		if sec, ok := jwtKV["jwt"]; ok {
			cfg.JWT.Secret = getStrWithEnvNested("JWT_SECRET", sec, "secret", "dev_secret")
			cfg.JWT.ExpiryMinutes = getIntWithEnvNested("JWT_EXPIRY_MINUTES", sec, "expiry_minutes", 60)
		} else {
			// плоская структура
			cfg.JWT.Secret = getStrWithEnv("JWT_SECRET", jwtKV, "secret", "dev_secret")
			cfg.JWT.ExpiryMinutes = getIntWithEnv("JWT_EXPIRY_MINUTES", jwtKV, "expiry_minutes", 60)
		}
	} else {
		cfg.JWT.Secret = getEnv("JWT_SECRET", "dev_secret")
		cfg.JWT.ExpiryMinutes = getEnvInt("JWT_EXPIRY_MINUTES", 60)
	}

	return cfg
}

// parseYAML — парсит простые YAML файлы без глубокой вложенности
// Формат: key: value (плоский) либо section: \n  key: value
func parseYAML(path string) (map[string]map[string]string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := map[string]map[string]string{}
	section := ""

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Проверяем, является ли строка началом секции (заканчивается на ':' без пробелов)
		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			section = strings.TrimSuffix(line, ":")
			if result[section] == nil {
				result[section] = map[string]string{}
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)

		// Если section не пустой, кладем в секцию, иначе в root
		if section != "" {
			if result[section] == nil {
				result[section] = map[string]string{}
			}
			result[section][key] = val
		} else {
			// плоская структура, создаем root секцию
			if result[""] == nil {
				result[""] = map[string]string{}
			}
			result[""][key] = val
		}
	}

	return result, sc.Err()
}

func getEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getStrWithEnv(envKey string, yaml map[string]map[string]string, key, def string) string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	if val, ok := yaml[""][key]; ok && val != "" {
		return val
	}
	return def
}

func getIntWithEnv(envKey string, yaml map[string]map[string]string, key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	if val, ok := yaml[""][key]; ok && val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return def
}

func getStrWithEnvNested(envKey string, section map[string]string, key, def string) string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	if val, ok := section[key]; ok && val != "" {
		return val
	}
	return def
}

func getIntWithEnvNested(envKey string, section map[string]string, key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	if val, ok := section[key]; ok && val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return def
}

// DSN возвращает строку подключения к БД
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// AMQPURL возвращает URL подключения к RabbitMQ
func (c MQConfig) AMQPURL() string {
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%d%s",
		c.User, c.Password, c.Host, c.Port, c.VHost,
	)
}
