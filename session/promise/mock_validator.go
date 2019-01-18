package promise

// TODO: remove when have a proper implementation
type NoopValidator struct{}

func (nv *NoopValidator) Validate(PromiseMessage) bool {
	return true
}
