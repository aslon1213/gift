package configs

import (
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	DB   *DBConfig
	Auth *AuthConfig
}

type DBConfig struct {
	URL            string
	Auth           bool
	MaxConnections uint64
	MinPoolSize    uint64
}
type AuthConfig struct {
	JwtSecret           string
	JwtExpiresIn        time.Duration
	JwtRefreshSecret    string
	JwtRefreshExpiresIn time.Duration
}

func LoadConfig(path string) (*Config, error) {
	err := godotenv.Load(
		".env",
	)
	if err != nil {
		fmt.Println("Error loading .env file", err)
	}
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.AddConfigPath(path)

	err = v.ReadInConfig()
	if err != nil {
		return nil, err
	}

	return &Config{
		DB: &DBConfig{
			URL:            v.GetString("db.url"),
			Auth:           v.GetBool("db.auth"),
			MaxConnections: v.GetUint64("db.max_connections"),
			MinPoolSize:    v.GetUint64("db.min_pool_size"),
		},
		Auth: &AuthConfig{
			JwtSecret:           v.GetString("auth.jwt_secret"),
			JwtExpiresIn:        v.GetDuration("auth.jwt_expires_in"),
			JwtRefreshSecret:    v.GetString("auth.jwt_refresh_secret"),
			JwtRefreshExpiresIn: v.GetDuration("auth.jwt_refresh_expires_in"),
		},
	}, nil
}
