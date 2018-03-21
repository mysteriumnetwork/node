package location

type resolverFake struct {
	country string
	error error
}

// NewResolverFake returns resolverFake which uses statically entered value
func NewResolverFake(country string) *resolverFake {
	return &resolverFake{
		country: country,
		error: nil,
	}
}

// NewFailingResolverFake returns resolverFake with entered error
func NewFailingResolverFake(err error) *resolverFake {
	return &resolverFake{
		country:  "",
		error:    err,
	}
}

// ResolveCountry maps given ip to country
func (d *resolverFake) ResolveCountry(ip string) (string, error) {
	return d.country, d.error
}
