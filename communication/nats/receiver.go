package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/nats-io/go-nats"

	"fmt"
	log "github.com/cihub/seelog"
)

const RECEIVER_LOG_PREFIX = "[NATS.Receiver] "

type receiverNats struct {
	connection   *nats.Conn
	messageTopic string
}

func (receiver *receiverNats) Receive(
	messageType communication.MessageType,
	listener communication.MessageListener,
) error {

	messageHandler := func(msg *nats.Msg) {
		err := listener.Message.Unpack(msg.Data)
		if err != nil {
			err = fmt.Errorf("Failed to unpack message '%s'. %s", messageType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		listener.Invoke()
	}

	_, err := receiver.connection.Subscribe(receiver.messageTopic+string(messageType), messageHandler)
	if err != nil {
		err = fmt.Errorf("Failed subscribe message '%s'. %s", messageType, err)
		return err
	}

	return nil
}

func (receiver *receiverNats) Respond(
	requestType communication.RequestType,
	handler communication.RequestHandler,
) error {

	messageHandler := func(msg *nats.Msg) {
		requestData := msg.Data
		log.Debug(RECEIVER_LOG_PREFIX, fmt.Sprintf("Request '%s' received: %s", requestType, requestData))

		err := handler.Request.Unpack(requestData)
		if err != nil {
			err = fmt.Errorf("Failed to unpack request '%s'. %s", requestType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		response := handler.Invoke()

		responseData, err := response.Pack()
		if err != nil {
			err = fmt.Errorf("Failed to pack response '%s'. %s", requestType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		err = receiver.connection.Publish(msg.Reply, responseData)
		if err != nil {
			err = fmt.Errorf("Failed to send response '%s'. %s", requestType, err)
			log.Error(RECEIVER_LOG_PREFIX, err)
			return
		}

		log.Debug(RECEIVER_LOG_PREFIX, fmt.Sprintf("Request '%s' response: %s", requestType, responseData))
	}

	_, err := receiver.connection.Subscribe(receiver.messageTopic+string(requestType), messageHandler)
	if err != nil {
		err = fmt.Errorf("Failed subscribe request '%s'. %s", requestType, err)
		return err
	}

	return nil
}
