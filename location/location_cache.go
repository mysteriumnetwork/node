package location

type locationCache struct {
	locationDetector Detector
	location Location
	err error
}

// NewLocationCache constructs Cache
func NewLocationCache(locationDetector Detector) Cache {
	return &locationCache{
		locationDetector: locationDetector,
	}
}

// Gets location from cache
func (lc *locationCache) Get() (Location, error) {
	return lc.location, lc.err
}

// Stores location to cache
func (lc *locationCache) RefreshAndGet() (Location, error)  {
	location, err := lc.locationDetector.DetectLocation()
	lc.location = location
	lc.err = err
	return lc.location, lc.err
}
