package nat

type NATService interface {
	Add(rule RuleForwarding)
	Start() error
	Stop() error
}

type RuleForwarding struct {
	SourceAddress string
	TargetIp      string
}
