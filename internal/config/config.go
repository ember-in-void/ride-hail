package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// parseFile — простой YAML-парсер вида "ключ: значение"
// callback получает (section, key, value).
func parseFile(path string, callback func(section, key, val string)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var section string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// поддержка ${VAR:-default}
		if strings.HasPrefix(val, "${") {
			val = expandEnv(val)
		}

		callback(section, key, val)
	}
	return scanner.Err()
}

func expandEnv(s string) string {
	s = strings.TrimPrefix(s, "${")
	s = strings.TrimSuffix(s, "}")
	parts := strings.SplitN(s, ":-", 2)
	envVal := os.Getenv(parts[0])
	if envVal != "" {
		return envVal
	}
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

type ServicesConfig struct {
	RideServicePort           int
	DriverLocationServicePort int
	AdminServicePort          int
}

func LoadServicesConfig(path string) (*ServicesConfig, error) {
	cfg := &ServicesConfig{}
	err := parseFile(path, func(section, key, val string) {
		switch key {
		case "ride_service":
			cfg.RideServicePort, _ = strconv.Atoi(val)
		case "driver_location_service":
			cfg.DriverLocationServicePort, _ = strconv.Atoi(val)
		case "admin_service":
			cfg.AdminServicePort, _ = strconv.Atoi(val)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("load services config: %w", err)
	}
	return cfg, nil
}

type WebSocketConfig struct {
	Port int
}

func LoadWSConfig(path string) (*WebSocketConfig, error) {
	cfg := &WebSocketConfig{}
	err := parseFile(path, func(section, key, val string) {
		if key == "port" {
			cfg.Port, _ = strconv.Atoi(val)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("load ws config: %w", err)
	}
	return cfg, nil
}

type RabbitMQConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

func LoadRabbitConfig(path string) (*RabbitMQConfig, error) {
	cfg := &RabbitMQConfig{}
	err := parseFile(path, func(section, key, val string) {
		switch key {
		case "host":
			cfg.Host = val
		case "port":
			cfg.Port, _ = strconv.Atoi(val)
		case "user":
			cfg.User = val
		case "password":
			cfg.Password = val
		}
	})
	if err != nil {
		return nil, fmt.Errorf("load rabbitmq config: %w", err)
	}
	return cfg, nil
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

func LoadDatabaseConfig(path string) (*DatabaseConfig, error) {
	cfg := &DatabaseConfig{}
	err := parseFile(path, func(section, key, val string) {
		switch key {
		case "host":
			cfg.Host = val
		case "port":
			cfg.Port, _ = strconv.Atoi(val)
		case "user":
			cfg.User = val
		case "password":
			cfg.Password = val
		case "database":
			cfg.Name = val
		}
	})
	if err != nil {
		return nil, fmt.Errorf("load db config: %w", err)
	}
	return cfg, nil
}

type JWT struct {
	Secret        string
	ExpiryMinutes int
}

func LoadJWTConfig(path string) (*JWT, error) {
	cfg := &JWT{}
	err := parseFile(path, func(section, key, val string) {
		if section != "jwt" {
			return
		}
		switch key {
		case "secret":
			cfg.Secret = val
		case "expiry_minutes":
			cfg.ExpiryMinutes, _ = strconv.Atoi(val)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("load jwt config: %w", err)
	}
	return cfg, nil
}
