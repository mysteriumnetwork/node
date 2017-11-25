package communication

import (
	"encoding/json"
	"reflect"
)

type JsonPayload struct {
	Model interface{}
}

func (payload JsonPayload) Pack() ([]byte, error) {
	return json.Marshal(payload.Model)
}

func (payload *JsonPayload) Unpack(data []byte) error {
	return json.Unmarshal(data, payload.Model)
}

func JsonPacker(model interface{}) *JsonPayload {
	return &JsonPayload{model}
}

func JsonListener(listener interface{}) MessageListener {
	listenerValue := reflect.ValueOf(listener)
	listenerType := parseCallbackType(listener)
	message := JsonPayload{
		Model: parseArgumentValue(listenerType).Interface(),
	}

	return MessageListener{
		Message: &message,
		Invoke: func() {
			listenerValue.Call([]reflect.Value{
				reflect.ValueOf(message.Model).Elem(),
			})
		},
	}
}

func JsonHandler(handler interface{}) RequestHandler {
	handlerValue := reflect.ValueOf(handler)
	handlerType := parseCallbackType(handler)
	request := JsonPayload{
		Model: parseArgumentValue(handlerType).Interface(),
	}
	response := JsonPayload{
		Model: parseReturnValue(handlerType).Interface(),
	}

	return RequestHandler{
		Request: &request,
		Invoke: func() Packer {
			handlerReturnValues := handlerValue.Call([]reflect.Value{
				reflect.ValueOf(request.Model).Elem(),
			})
			response.Model = handlerReturnValues[0].Interface()

			return response
		},
	}
}

func parseCallbackType(callback interface{}) reflect.Type {
	callbackType := reflect.TypeOf(callback)
	if callbackType.Kind() != reflect.Func {
		panic("Callback needs to be a func")
	}

	return callbackType
}

func parseArgumentValue(callbackType reflect.Type) reflect.Value {
	if callbackType.NumIn() != 1 {
		panic("Callback should accept 1 argument")
	}

	return reflect.New(callbackType.In(0))
}

func parseReturnValue(callbackType reflect.Type) reflect.Value {
	if callbackType.NumOut() != 1 {
		panic("Callback should return 1 argument")
	}

	return reflect.New(callbackType.Out(0))
}
