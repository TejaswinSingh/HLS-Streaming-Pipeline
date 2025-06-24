package trip

import (
	"fmt"
	"path/filepath"
	"time"
)

type ArchiveTripStorage interface {
	GetTrip(date time.Time, schoolID, busID string, tripID string) ([]map[string]any, error)
	GetBusTrips(date time.Time, schoolID, busID string) ([]map[string]any, error)
	GetSchoolTrips(date time.Time, schoolID string) ([]map[string]any, error)
	MakeTripPath(tm time.Time, schoolID, busID, tripID string) string
	StartArchiver(interval time.Duration)
	StopArchiver()
	MoveAll()
	MoveToArchive(trip *Trip) error
}

type S3TripStorage struct {
	basePath string
	// done     chan struct{}
	// s3 config
}

func (s *S3TripStorage) MakeTripPath(tm time.Time, schoolID, busID, tripID string) string {
	date := tm.Format(YYYYMMDD)
	return filepath.Join(s.basePath, schoolID, busID, date, fmt.Sprintf("%s.json", tripID))
}

// func (s *MongoTripStorage) StartArchiver(interval time.Duration) {
// 	go func() {
// 		ticker := time.NewTicker(interval)
// 		defer ticker.Stop()
// 		for {
// 			select {
// 			case <-ticker.C:
// 				s.MoveAll()
// 			case <-s.done:
// 				return
// 			}
// 		}
// 	}()
// }

// func (s *MongoTripStorage) StopArchiver() {
// 	close(s.done)
// }
