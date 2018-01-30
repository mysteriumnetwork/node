package session

import (
	"github.com/mysterium/node/communication"
)

// NewDialogHandler constructs handler which gets all incoming dialogs and starts handling them
func NewDialogHandler(proposalId int, sessionManager Manager) *handler {
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

// Handle starts serving services in given Dialog instance
func (handler *handler) Handle(dialog communication.Dialog) error {
	subscribeError := dialog.Respond(handler.sessionCreateConsumer)
	if subscribeError != nil {
		return subscribeError
	}

	return nil
}
