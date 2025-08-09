package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	mongodb "github.com/Resolyte-official/service-gps/pkg/db/mongo"
)

// Configuration .
type Configuration struct {
	Env                GoEnv
	HttpMode           HttpMode
	HttpPort           string
	CorsOrigins        []string
	GovernanceGrpcAddr string
	Mongo              mongodb.MongoConfiguration
	S3BucketName       string
}

const (
	defaultCORSOrigin = ""
	defaultHTTPPort   = "12003"
	defaultMongoHost  = "mongo"
	defaultMongoPort  = "27017"
)

// NewEnvironmentConfiguration creates a configuration populated from os environment variables.
func NewEnvironmentConfiguration() Configuration {

	var httpMode HttpMode
	if envBool("USE_HTTPS", false) {
		httpMode = Https
	}

	return Configuration{
		Env:                ValidateEnv(GoEnv(os.Getenv("ENV"))),
		HttpMode:           ValidateHttpMode(httpMode),
		HttpPort:           envString("GPS_HTTP_PORT", defaultHTTPPort),
		CorsOrigins:        parseStringsEnv("CORS_ORIGINS", []string{defaultCORSOrigin}),
		GovernanceGrpcAddr: requireEnv("GOVERNANCE_GRPC_ADDR"),
		S3BucketName:       requireEnv("S3_BUCKET_NAME"),
		Mongo: mongodb.MongoConfiguration{
			Host:     envString("MONGO_HOST", defaultMongoHost),
			Port:     envString("MONGO_PORT", defaultMongoPort),
			Username: requireEnv("MONGODB_INITDB_ROOT_USERNAME"),
			Password: requireEnv("MONGODB_INITDB_ROOT_PASSWORD"),
		},
	}
}

func envString(key, fallback string) string {
	e, ok := os.LookupEnv(key)
	val := strings.TrimSpace(e)
	if !ok || val == "" {
		return fallback
	}
	return val
}

func envBool(key string, fallback bool) bool {
	val, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(key)))
	if err != nil {
		return fallback
	}
	return val
}

func requireEnv(key string) string {
	val, ok := os.LookupEnv(key)
	val = strings.TrimSpace(val)
	if !ok || val == "" {
		panic(fmt.Errorf("invalid environment configuration: %s is required", key))
	}
	return val
}

func parseStringsEnv(key string, fallback []string) []string {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback
	}
	parts := strings.Split(val, ",")
	var strs []string
	for _, o := range parts {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			strs = append(strs, trimmed)
		}
	}
	return strs
}
