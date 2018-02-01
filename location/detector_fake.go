package location

type detectorFake struct {
	country string
}

// NewDetectorFake returns Detector which uses statically entered value
func NewDetectorFake(country string) *detectorFake {
	return &detectorFake{country: country}
}

// DetectCountry maps given ip to country
func (d *detectorFake) DetectCountry(ip string) (string, error) {
	return d.country, nil
}
