package session

import (
	"github.com/mysterium/node/communication"
)

// NewDialogHandler constructs handler which get all incoming dialogs does starts handling them
func NewDialogHandler(proposalId int, sessionManager ManagerInterface) *handler {
	return &handler{
		sessionCreateConsumer: &SessionCreateConsumer{
			CurrentProposalID: proposalId,
			SessionManager:    sessionManager,
		},
	}
}

type handler struct {
	sessionCreateConsumer communication.RequestConsumer
}

func (handler *handler) Handle(dialog communication.Dialog) error {
	subscribeError := dialog.Respond(handler.sessionCreateConsumer)
	if subscribeError != nil {
		return subscribeError
	}

	return nil
}
