package event

const Topic = "Session change"

type Action string

var (
	Created Action = "Created"
	Removed Action = "Removed"
)

type Payload struct {
	Action Action
	ID     string
}
