package config

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Version  string
	LogLevel string
	LogMode  string
	Mode     string

	Queue struct {
		Memory struct {
			Enabled bool
			Size    int
		}
		Nats struct {
			Host     string
			Port     int
			Username string
			Password string
		}
	}

	Reddit struct {
		Agent        string
		ClientID     string
		ClientSecret string
		Username     string
		Password     string
		SubReddits   []string
	}
	Database struct {
		Memory struct {
			Enabled bool
		}
		Pg struct {
			Enabled  bool
			Host     string
			Port     int
			Database string
			Username string
			Password string
		}
	}
	Data struct {
		XetraCSV string
	}
	HTTPServer struct {
		Port              int
		BasicAuthDisabled bool
		Username          string
		Password          string
	}
}

func LoadConfigFromEnv(version string) Config {
	c := Config{
		Version: version,
	}

	c.LogLevel = fromEnvStr("LOG_LEVEL", "debug")
	c.LogMode = fromEnvStr("LOG_MODE", "develop")

	c.Mode = fromEnvStr("MODE", "prod")

	c.Queue.Memory.Size = fromEnvInt("QUEUE_LENGTH", 20)

	c.Reddit.Agent = fromEnvStr("REDDIT_AGENT", "")
	c.Reddit.ClientID = fromEnvStr("REDDIT_CLIENT_ID", "")
	c.Reddit.ClientSecret = fromEnvStr("REDDIT_CLIENT_SECRET", "")
	c.Reddit.Username = fromEnvStr("REDDIT_USERNAME", "")
	c.Reddit.Password = fromEnvStr("REDDIT_PASSWORD", "")
	subs := fromEnvStr("REDDIT_SUBREDDITS", "")
	c.Reddit.SubReddits = strings.Split(subs, ",")

	c.Queue.Nats.Host = fromEnvStr("QUEUE_NATS_HOST", "localhost")
	c.Queue.Nats.Port = fromEnvInt("QUEUE_NATS_PORT", 4222)
	c.Queue.Nats.Username = fromEnvStr("QUEUE_NATS_USERNAME", "mswkn")
	c.Queue.Nats.Password = fromEnvStr("QUEUE_NATS_PASSWORD", "mswkn")

	c.Database.Memory.Enabled = fromEnvBool("DATABASE_MEMORY_ENABLED", true)
	c.Database.Pg.Enabled = fromEnvBool("DATABASE_PG_ENABLED", false)
	c.Database.Pg.Host = fromEnvStr("DATABASE_PG_HOST", "localhost")
	c.Database.Pg.Port = fromEnvInt("DATABASE_PG_PORT", 5432)
	c.Database.Pg.Database = fromEnvStr("DATABASE_PG_DATABASE", "mswkn")
	c.Database.Pg.Username = fromEnvStr("DATABASE_PG_USERNAME", "mswkn")
	c.Database.Pg.Password = fromEnvStr("DATABASE_PG_PASSWORD", "mswkn")

	c.Data.XetraCSV = fromEnvStr("DATA_XETRA_CSV", "")

	c.HTTPServer.Port = fromEnvInt("HTTP_SERVER_PORT", 3000)
	c.HTTPServer.BasicAuthDisabled = fromEnvBool("HTTP_SERVER_AUTH_DISABLED", false)
	c.HTTPServer.Username = fromEnvStr("HTTP_SERVER_USERNAME", "")
	c.HTTPServer.Password = fromEnvStr("HTTP_SERVER_PASSWORD", "")

	if os.Getenv("DEBUG_CONFIG_DUMP") == "1" {
		log.Printf("config: %+v", c)
	}

	return c
}

func fromEnvStr(name, fallback string) string {
	val, isSet := os.LookupEnv(name)
	//	log.Printf("name [%s] val [%s] isset[%s]", name, val, isSet)
	if !isSet {
		return fallback
	}
	return val
}

func fromEnvInt(name string, fallback int) int {
	val, isSet := os.LookupEnv(name)
	//	log.Printf("name [%s] val [%s] isset[%s]", name, val, isSet)
	if !isSet {
		return fallback
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		panic(fmt.Sprintf("the value '%s' could not be parsed to an int value for %s", val, name))
	}
	return intVal
}

func fromEnvBool(name string, fallback bool) bool {
	val, isSet := os.LookupEnv(name)
	if !isSet {
		return fallback
	}

	if val == "" || val == "false" || val == "0" {
		return false
	}

	if val == "1" || val == "true" {
		return true
	}

	return false
}

//
//func fromEnvDuration(name string, fallback time.Duration) time.Duration {
//	val, isSet := os.LookupEnv(name)
//
//	if !isSet {
//		return fallback
//	}
//
//	d, err := time.ParseDuration(val)
//	if err != nil {
//		log.Warn().Str("value", val).Msg("could not parse env value to time.Duration")
//		return fallback
//	}
//
//	return d
//}
