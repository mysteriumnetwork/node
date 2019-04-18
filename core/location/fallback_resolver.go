package location

import (
	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
)

const fallbackResolverLogPrefix = "[fallback-resolver] "

// ErrLocationResolutionFailed represents a failure to resolve location and running out of fallbacks to try
var ErrLocationResolutionFailed = errors.New("location resolution failed")

// FallbackResolver represents a resolver that tries multiple resolution techniques in sequence until one of them completes successfully, or all of them fail.
type FallbackResolver struct {
	LocationResolvers []LocationResolver
}

// LocationResolver allows us to detect location
type LocationResolver interface {
	DetectLocation() (Location, error)
}

// NewFallbackResolver returns a new instance of fallback resolver
func NewFallbackResolver(resolvers []LocationResolver) *FallbackResolver {
	return &FallbackResolver{
		LocationResolvers: resolvers,
	}
}

// DetectLocation allows us to detect our current location
func (fr *FallbackResolver) DetectLocation() (Location, error) {
	for _, v := range fr.LocationResolvers {
		loc, err := v.DetectLocation()
		if err != nil {
			log.Warn(fallbackResolverLogPrefix, "could not resolve location", err)
		} else {
			return loc, err
		}
	}
	return Location{}, ErrLocationResolutionFailed
}
