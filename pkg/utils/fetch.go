package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/Resolyte-official/service-gps/pkg/logging"
	"github.com/Resolyte-official/service-gps/pkg/trip"
)

type DeviceTrip struct {
	GpsTrackerID string
	TripID       string
}

type FetchList struct {
	mu            sync.Mutex
	lw            logging.LogWriter
	dbTripStorage trip.DBTripStorage
	list          []DeviceTrip
}

func NewFetchList(lw logging.LogWriter, ts trip.DBTripStorage) *FetchList {
	return &FetchList{
		mu:            sync.Mutex{},
		lw:            lw,
		dbTripStorage: ts,
		list:          make([]DeviceTrip, 0),
	}
}

func (fl *FetchList) Add(dt DeviceTrip) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.list = append(fl.list, dt)
}

func (fl *FetchList) All() []DeviceTrip {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	copied := make([]DeviceTrip, len(fl.list))
	copy(copied, fl.list)
	return copied
}

func (fl *FetchList) Remove(gpsTrackerID string) (*DeviceTrip, bool) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	for i, dt := range fl.list {
		if dt.GpsTrackerID == gpsTrackerID {
			fl.list = slices.Delete(fl.list, i, i+1)
			return &dt, true
		}
	}
	return nil, false
}

func (fl *FetchList) FetchAndAppendCoordinates() {
	// TODO: test with real gps devices

	logger := logging.NewContextLogger(fl.lw, "GpsFetchRoutine")
	var wg sync.WaitGroup

	devices := fl.All()
	if len(devices) == 0 {
		logger.Info("no devices in fetch list")
		return
	}

	for _, dt := range devices {

		wg.Add(1)
		go func(dt DeviceTrip) {
			defer wg.Done()

			// ping gps device
			coord, err := fetchCoordinates(dt.GpsTrackerID)
			if err != nil {
				logger.Warnf("failed to fetch coordinates for device [%s] | err: %s", dt.GpsTrackerID, err)
				return
			}

			// append coordinates to trip in mongodb
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			if err := fl.dbTripStorage.AppendCoordinate(ctx, dt.TripID, *coord); err != nil {
				logger.Warnf("failed to append coordinates for device [%s] trip [%s] | err: %s", dt.GpsTrackerID, dt.TripID, err)
			}

			logger.Infof("coordinates appended for device [%s] trip [%s]", dt.GpsTrackerID, dt.TripID)
		}(dt)
	}
	wg.Wait() // waits for all fetch-and-append's to finish
}

func fetchCoordinates(gps_id string) (*trip.Coordinate, error) {
	url := fmt.Sprintf("http://localhost:8080/get-location?gps_id=%s", gps_id)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%d response from GPS device", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var coord trip.Coordinate
	if err := json.Unmarshal(body, &coord); err != nil {
		return nil, err
	}

	coord.Timestamp = time.Now().UTC() // add timestamp if gps device doesnt send one
	return &coord, nil
}
