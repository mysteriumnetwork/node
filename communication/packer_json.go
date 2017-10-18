package communication

import (
	"encoding/json"
	"reflect"
)

type JsonPayload struct {
	Model interface{}
}

func (payload JsonPayload) Pack() (data []byte) {
	data, err := json.Marshal(payload.Model)
	if err != nil {
		panic(err)
	}
	return []byte(data)
}

func (payload *JsonPayload) Unpack(data []byte) {
	err := json.Unmarshal(data, payload.Model)
	if err != nil {
		panic(err)
	}
}

func JsonListener(listener interface{}) MessageListener {
	listenerValue := reflect.ValueOf(listener)
	listenerType := parseCallbackType(listener)
	message := JsonPayload{
		Model: parseArgumentValue(listenerType).Interface(),
	}

	return func(messageData []byte) {
		message.Unpack(messageData)

		listenerValue.Call([]reflect.Value{
			reflect.ValueOf(message.Model).Elem(),
		})
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

	return func(requestData []byte) []byte {
		request.Unpack(requestData)

		handlerReturnValues := handlerValue.Call([]reflect.Value{
			reflect.ValueOf(request.Model).Elem(),
		})

		response.Model = handlerReturnValues[0].Interface()
		return response.Pack()
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
