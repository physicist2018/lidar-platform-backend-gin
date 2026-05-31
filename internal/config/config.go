package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	ServerAddress   string        `mapstructure:"SERVER_ADDRESS"`
	DBSource        string        `mapstructure:"DB_SOURCE"`
	RedisAddress    string        `mapstructure:"REDIS_ADDRESS"`
	RedisPassword   string        `mapstructure:"REDIS_PASSWORD"`
	RedisDB         int           `mapstructure:"REDIS_DB"`
	CacheTTLDefault time.Duration `mapstructure:"CACHE_TTL_DEFAULT"`
	JWTSecret       string        `mapstructure:"JWT_SECRET"`
	JWTExpiration   time.Duration `mapstructure:"JWT_EXPIRATION"`
	MaxWorkers      int           `mapstructure:"MAX_WORKERS"`
	MinioEndpoint   string        `mapstructure:"MINIO_ENDPOINT"`
	MinioAccessKey  string        `mapstructure:"MINIO_ACCESS_KEY"`
	MinioSecretKey  string        `mapstructure:"MINIO_SECRET_KEY"`
	MinioBucket     string        `mapstructure:"MINIO_BUCKET"`
	MinioUseSSL     bool          `mapstructure:"MINIO_USE_SSL"`
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
