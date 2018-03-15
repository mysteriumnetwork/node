package location

type resolverFake struct {
	country string
}

// NewResolverFake returns Resolver which uses statically entered value
func NewResolverFake(country string) *resolverFake {
	return &resolverFake{country: country}
}

// ResolveCountry maps given ip to country
func (d *resolverFake) ResolveCountry(ip string) (string, error) {
	return d.country, nil
}
