package gps

import (
	"context"
	"net/http"

	"github.com/Resolyte-official/service-gps/pkg/config"
)

type gpsService struct {
	envConfig *config.Configuration
}

func NewService(envConfig *config.Configuration) Service {
	return &gpsService{
		envConfig: envConfig,
	}
}

func (s *gpsService) ServiceStatus(_ context.Context) (int, error) {
	return http.StatusOK, nil
}
