package communication

import (
	"encoding/json"
	"github.com/pkg/errors"
	"reflect"
)

func JsonPacker(message interface{}) Packer {
	return func() []byte {
		data, err := json.Marshal(message)
		if err != nil {
			panic(err)
		}

		return data
	}
}

func JsonUnpacker(messagePtr interface{}) Unpacker {
	return func(data []byte) {
		err := json.Unmarshal(data, &messagePtr)
		if err != nil {
			panic(err)
		}
	}
}

func JsonListener(listener interface{}) MessageListener {
	listenerValue := reflect.ValueOf(listener)
	messageValue, err := parseArgumentValue(listener)
	if err != nil {
		panic(err)
	}

	return func(messageData []byte) {
		JsonUnpacker(messageValue.Interface())(messageData)

		listenerValue.Call([]reflect.Value{
			reflect.Indirect(messageValue),
		})
	}
}

func JsonHandler(handler interface{}) RequestHandler {
	handlerValue := reflect.ValueOf(handler)
	requestValue, err := parseArgumentValue(handler)
	if err != nil {
		panic(err)
	}

	return func(requestData []byte) []byte {
		JsonUnpacker(requestValue.Interface())(requestData)

		handlerResult := handlerValue.Call([]reflect.Value{
			reflect.Indirect(requestValue),
		})
		response := handlerResult[0]

		return JsonPacker(response.Interface())()
	}
}

func parseArgumentValue(callback interface{}) (argumentValue reflect.Value, err error) {
	callbackType := reflect.TypeOf(callback)

	if callbackType.Kind() != reflect.Func {
		err = errors.New("Message callback needs to be a func")
		return
	}
	if callbackType.NumIn() != 1 {
		err = errors.New("Message callback should accept message")
		return
	}

	argumentValue = reflect.New(callbackType.In(0))
	return
}
