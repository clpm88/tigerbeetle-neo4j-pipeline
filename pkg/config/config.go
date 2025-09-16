package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the services
type Config struct {
	TigerBeetle TigerBeetleConfig
	Redpanda    RedpandaConfig
	Neo4j       Neo4jConfig
}

// TigerBeetleConfig holds TigerBeetle connection settings
type TigerBeetleConfig struct {
	Address string
}

// RedpandaConfig holds Redpanda/Kafka connection settings
type RedpandaConfig struct {
	Brokers     []string
	Topic       string
	ConsumerGroup string
}

// Neo4jConfig holds Neo4j connection settings
type Neo4jConfig struct {
	URI      string
	Username string
	Password string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		TigerBeetle: TigerBeetleConfig{
			Address: getEnv("TIGERBEETLE_ADDRESS", "3000"),
		},
		Redpanda: RedpandaConfig{
			Brokers:       []string{getEnv("REDPANDA_BROKERS", "localhost:19092")},
			Topic:         getEnv("REDPANDA_TOPIC", "transactions"),
			ConsumerGroup: getEnv("REDPANDA_CONSUMER_GROUP", "neo4j-sink-group"),
		},
		Neo4j: Neo4jConfig{
			URI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
			Username: getEnv("NEO4J_USERNAME", "neo4j"),
			Password: getEnv("NEO4J_PASSWORD", "password"),
		},
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as an integer with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}