package conf

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type DBSettings struct {
	DbHost            string `env:"DATABASE_HOST,required"`
	DbName            string `env:"DATABASE_NAME,required"`
	DbPassword        string `env:"DATABASE_PASSWORD,required"`
	DbPort            string `env:"DATABASE_PORT,required"`
	DbUser            string `env:"DATABASE_USER,required"`
	DbMaxIdleConns    int    `env:"DATABASE_MAX_IDLE_CONNS" envDefault:"50"`
	DbMaxOpenConns    int    `env:"DATABASE_MAX_OPEN_CONNS" envDefault:"50"`
	DbConnMaxLifetime int    `env:"DATABASE_CONN_MAX_LIFETIME" envDefault:"5"`
	DbConnMaxIdleTime int    `env:"DATABASE_CONN_MAX_IDLE_TIME" envDefault:"15"`
	DbSchema          string `env:"DATABASE_SCHEMA" envDefault:"public"`
}

func LoadDBSettings() (DBSettings, error) {
	cfg := DBSettings{}

	opts := env.Options{OnSet: OnSetConfig}
	if err := env.ParseWithOptions(&cfg, opts); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func DBUriPsql(dbSetting DBSettings) string {
	databaseUrl := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?search_path=%s&sslmode=disable",
		dbSetting.DbUser,
		dbSetting.DbPassword,
		dbSetting.DbHost,
		dbSetting.DbPort,
		dbSetting.DbName,
		dbSetting.DbSchema)

	return databaseUrl
}
