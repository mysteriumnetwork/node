package validators

import "github.com/mysteriumnetwork/node/session/promise"

// TODO: remove when have a proper implementation
type NoopValidator struct{}

func (nv *NoopValidator) Validate(message promise.PromiseMessage) bool {
	return true
}
