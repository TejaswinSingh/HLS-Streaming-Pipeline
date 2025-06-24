package gps

import (
	"context"
	"net/http"

	"github.com/Resolyte-official/service-gps/pkg/config"
	mongodb "github.com/Resolyte-official/service-gps/pkg/db/mongo"
	"github.com/Resolyte-official/service-gps/pkg/trip"
)

type gpsService struct {
	envConfig     *config.Configuration
	mongoSvc      *mongodb.MongoService
	dbTripStorage trip.DBTripStorage
}

func NewService(envConfig *config.Configuration, mongoSvc *mongodb.MongoService, dbTripStorage trip.DBTripStorage) Service {
	return &gpsService{
		envConfig:     envConfig,
		mongoSvc:      mongoSvc,
		dbTripStorage: dbTripStorage,
	}
}

func (s *gpsService) ServiceStatus(_ context.Context) (int, error) {
	return http.StatusOK, nil
}
