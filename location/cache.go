package location

type cache struct {
	location Location
}

// NewLocationCache constructs Cache
func NewLocationCache() Cache {
	return &cache{}
}

// Gets location from cache
func (lc *cache) Get() Location {
	return lc.location
}

// Stores location to cache
func (lc *cache) Set(location Location) {
	lc.location = location
}
