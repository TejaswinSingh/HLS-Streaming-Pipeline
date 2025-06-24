package utils

import (
	"fmt"
	"slices"
	"sync"
)

type DeviceTrip struct {
	GpsTrackerID string
	TripID       string
}

type FetchList struct {
	mu   sync.Mutex
	list []DeviceTrip
}

func NewFetchList() *FetchList {
	return &FetchList{
		mu:   sync.Mutex{},
		list: make([]DeviceTrip, 0),
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

func (fl *FetchList) FetchAndAppendCoordinates() error {
	// TODO: Implement
	for _, id := range fl.All() {
		fmt.Printf("Fetching coordinates for device ID: %s", id)
	}
	return nil
}
