package communication

type RequestType string

type RequestConsumer interface {
	ConsumeRequest(requestBody []byte) (responseBody []byte)
}
