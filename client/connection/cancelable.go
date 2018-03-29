package connection

import "errors"

type blockingF func() (interface{}, error)

type cleanupF func(interface{}, error)

type cancelChannel chan int

type actionResult struct {
	val interface{}
	err error
}

type actionResultChannel chan actionResult

var errActionCancelled = errors.New("request was cancelled")

var errUndefinedRequest = errors.New("undefined request called")

func noCleanup(interface{}, error) {

}

func undefinedRequest() (interface{}, error) {
	return nil, errUndefinedRequest
}

func skipOnError(callback func(interface{})) cleanupF {
	return func(val interface{}, err error) {
		if err == nil {
			callback(val)
		}
	}
}

type cancelableAction struct {
	cancelled     cancelChannel
	requestAction blockingF
	cleanupAction cleanupF
	cancelAction  func()
}

func newCancelable() cancelableAction {
	channel := make(cancelChannel, 1)
	return cancelableAction{
		cancelled:     channel,
		requestAction: undefinedRequest,
		cleanupAction: noCleanup,
		cancelAction: callOnce(func() {
			close(channel)
		}),
	}
}

func (c cancelableAction) request(method blockingF) cancelableAction {
	c.requestAction = method
	return c
}

func (c cancelableAction) cleanup(cleanup cleanupF) cancelableAction {
	c.cleanupAction = cleanup
	return c
}

func (c cancelableAction) call() (interface{}, error) {
	return callBlockingAction(c.requestAction, c.cleanupAction, c.cancelled)
}

func (c cancelableAction) cancel() {
	c.cancelAction()
}

func callBlockingAction(method blockingF, cleanup cleanupF, cancel cancelChannel) (interface{}, error) {

	resultChannel := make(actionResultChannel, 1)

	go func() {
		val, err := method()
		resultChannel <- actionResult{val, err}
	}()

	select {
	case res := <-resultChannel:
		return res.val, res.err
	case <-cancel:
		go func() {
			res := <-resultChannel
			cleanup(res.val, res.err)
		}()
		return nil, errActionCancelled
	}
}
