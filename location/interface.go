package location

// Resolver allows resolving location by ip
type Resolver interface {
	ResolveCountry(ip string) (string, error)
}

type LocationDetector interface {
	DetectCountry() (string, error)
}