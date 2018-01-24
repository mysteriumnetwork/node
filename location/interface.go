package location

// Detector allows detecting location
type Detector interface {
	DetectCountry(ip string) (string, error)
}
