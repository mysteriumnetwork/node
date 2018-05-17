package location

type cache struct {
	locationDetector Detector
	location         Location
}

// NewLocationCache constructs Cache
func NewLocationCache(locationDetector Detector) Cache {
	return &cache{
		locationDetector: locationDetector,
	}
}

// Gets location from cache
func (lc *cache) Get() Location {
	return lc.location
}

// Stores location to cache
func (lc *cache) RefreshAndGet() (Location, error) {
	location, err := lc.locationDetector.DetectLocation()
	lc.location = location
	return lc.location, err
}
