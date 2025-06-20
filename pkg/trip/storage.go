package trip

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	svcErrors "github.com/Resolyte-official/service-gps/pkg/errors"
)

type BufferedTripStorage interface {
	CreateTrip(path string) error
	EndTrip(path string) error
	// AddCoordinate(path string, coord Coordinate) error
	GetTrip(path string) (*Trip, error)
	readTripLocked(path string) (*Trip, error)
	UpdateTrip(path string, newTrip *Trip) error
	writeTripLocked(path string, newTrip *Trip) error
	DeleteTrip(path string) error
	ListTrips(date time.Time, schoolID, busID string) ([]string, error)

	// sync map utilities
	getLock(path string) *sync.RWMutex
	clearLock(path string)

	// buffer methods
	BufferCoordinate(path string, coord Coordinate) error
	StartFlusher(interval time.Duration)
	flushAll()
	flushToDisk(path string, coords []Coordinate) error
}

type JSONBufferedTripStorage struct {
	basePath          string
	muBuffer          sync.RWMutex            // mutex for buffer operations
	tripCoordsBuffers map[string][]Coordinate // key: trip-path, val: coordinates
	locks             sync.Map                // map to hold locks for each trip file
	done              chan struct{}
}

func NewBufferedTripStorage(basePath string) BufferedTripStorage {
	return &JSONBufferedTripStorage{
		basePath:          basePath,
		muBuffer:          sync.RWMutex{},
		tripCoordsBuffers: make(map[string][]Coordinate),
		locks:             sync.Map{},
		done:              make(chan struct{}),
	}
}

func (s *JSONBufferedTripStorage) getLock(path string) *sync.RWMutex {
	lock, _ := s.locks.LoadOrStore(path, &sync.RWMutex{})
	return lock.(*sync.RWMutex)
}

func (s *JSONBufferedTripStorage) clearLock(path string) {
	s.locks.Delete(path)
}

func (s *JSONBufferedTripStorage) TripPath(date time.Time, schoolID, busID, tripID string) string {
	dateStr := date.Format(YYYYMMDD)
	return filepath.Join(s.basePath, dateStr, schoolID, busID, fmt.Sprintf("%s.json", tripID))
}

func (s *JSONBufferedTripStorage) CreateTrip(path string) error {
	lock := s.getLock(path)
	lock.Lock()
	defer lock.Unlock()

	if _, err := os.Stat(path); err == nil { // file already exists
		return svcErrors.ErrInternalServer.WithInternalErr(err)
	}

	if err := os.MkdirAll(filepath.Dir(path), RWX_R_R); err != nil {
		return svcErrors.ErrInternalServer.WithInternalErr(err)
	}

	trip := Trip{
		StartTime:   time.Now().UTC(),
		Coordinates: []Coordinate{},
	}

	data, err := json.MarshalIndent(trip, "", "  ")
	if err != nil {
		return svcErrors.ErrInternalServer.WithInternalErr(err)
	}

	if err := os.WriteFile(path, data, RW_R_R); err != nil {
		return svcErrors.ErrInternalServer.WithInternalErr(err)
	}

	s.muBuffer.Lock()
	defer s.muBuffer.Unlock()
	s.tripCoordsBuffers[path] = make([]Coordinate, 0) // Initialize coord buffer for this trip

	return nil
}

func (s *JSONBufferedTripStorage) EndTrip(path string) error {
	lock := s.getLock(path)
	lock.Lock()
	defer lock.Unlock()

	trip, err := s.readTripLocked(path)
	if err != nil {
		return err
	}

	// fetch any unflushed coordinates inside the buffer before deleting it
	s.muBuffer.Lock()
	defer s.muBuffer.Unlock()
	coords := s.tripCoordsBuffers[path]

	// update trip with buffered coordinates and endtime
	trip.Coordinates = append(trip.Coordinates, coords...)
	trip.EndTime = time.Now().UTC()
	if err := s.writeTripLocked(path, trip); err != nil {
		return err
	}

	delete(s.tripCoordsBuffers, path) // clear the buffer for this trip after write was successful

	return nil
}

func (s *JSONBufferedTripStorage) GetTrip(path string) (*Trip, error) {
	lock := s.getLock(path)
	lock.RLock()
	defer lock.RUnlock()

	return s.readTripLocked(path)
}

// Note: assumes mutex is already acquired by the caller
func (s *JSONBufferedTripStorage) readTripLocked(path string) (*Trip, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, svcErrors.ErrInternalServer.WithInternalErr(err)
	}

	var trip Trip
	if err := json.Unmarshal(data, &trip); err != nil {
		return nil, svcErrors.ErrInternalServer.WithInternalErr(err)
	}
	return &trip, nil
}

func (s *JSONBufferedTripStorage) UpdateTrip(path string, newTrip *Trip) error {
	lock := s.getLock(path)
	lock.Lock()
	defer lock.Unlock()

	return s.writeTripLocked(path, newTrip)
}

// Note: assumes mutex is already acquired by the caller
func (s *JSONBufferedTripStorage) writeTripLocked(path string, newTrip *Trip) error {
	data, err := json.MarshalIndent(newTrip, "", "  ")
	if err != nil {
		return svcErrors.ErrInternalServer.WithInternalErr(err)
	}
	if err := os.WriteFile(path, data, RW_R_R); err != nil {
		return svcErrors.ErrInternalServer.WithInternalErr(err)
	}

	return nil
}

func (s *JSONBufferedTripStorage) DeleteTrip(path string) error {
	lock := s.getLock(path)
	lock.Lock()
	defer s.clearLock(path)
	defer lock.Unlock()

	if err := os.Remove(path); err != nil {
		return svcErrors.ErrInternalServer.WithInternalErr(err)
	}

	return nil
}

func (s *JSONBufferedTripStorage) ListTrips(date time.Time, schoolID, busID string) ([]string, error) {
	dateStr := date.Format(YYYYMMDD)
	dir := filepath.Join(s.basePath, dateStr, schoolID, busID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, svcErrors.ErrInternalServer.WithInternalErr(err)
	}

	var trips []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			trips = append(trips, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return trips, nil
}

func (s *JSONBufferedTripStorage) BufferCoordinate(path string, coord Coordinate) error {
	s.muBuffer.Lock()
	defer s.muBuffer.Unlock()

	if _, ok := s.tripCoordsBuffers[path]; !ok {
		return svcErrors.ErrTripNotFound
	}

	s.tripCoordsBuffers[path] = append(s.tripCoordsBuffers[path], coord)
	return nil
}

func (s *JSONBufferedTripStorage) StartFlusher(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.flushAll()
			case <-s.done:
				return
			}
		}
	}()
}

func (s *JSONBufferedTripStorage) StopFlusher() {
	close(s.done)
}

func (s *JSONBufferedTripStorage) flushAll() {
	s.muBuffer.Lock()
	defer s.muBuffer.Unlock()

	for path, coords := range s.tripCoordsBuffers {
		if len(coords) == 0 {
			continue
		}
		if err := s.flushToDisk(path, coords); err == nil {
			s.tripCoordsBuffers[path] = make([]Coordinate, 0) // reset buffer
		} else {
			// keep original buffer if flush failed
			// TODO: log error
		}
	}
}

func (s *JSONBufferedTripStorage) flushToDisk(path string, coords []Coordinate) error {
	lock := s.getLock(path)
	lock.Lock()
	defer lock.Unlock()

	trip, err := s.readTripLocked(path)
	if err != nil {
		return err
	}

	trip.Coordinates = append(trip.Coordinates, coords...)
	if err := s.writeTripLocked(path, trip); err != nil {
		return err
	}

	return nil
}
