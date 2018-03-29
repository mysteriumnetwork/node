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

var errActionCancelled = errors.New("action was cancelled")

var errUndefinedAction = errors.New("undefined action called")

func noCleanup(interface{}, error) {

}

func undefinedAction() (interface{}, error) {
	return nil, errUndefinedAction
}

func skipOnError(callback func(interface{})) cleanupF {
	return func(val interface{}, err error) {
		if err == nil {
			callback(val)
		}
	}
}

type cancelable struct {
	cancelChannel  cancelChannel
	blockingAction blockingF
	cleanupAction  cleanupF
	cancelAction   func()
}

func newCancelable() cancelable {
	channel := make(cancelChannel, 1)
	return cancelable{
		cancelChannel:  channel,
		blockingAction: undefinedAction,
		cleanupAction:  noCleanup,
		cancelAction: callOnce(func() {
			close(channel)
		}),
	}
}

func (c cancelable) action(method blockingF) cancelable {
	c.blockingAction = method
	return c
}

func (c cancelable) cleanup(cleanup cleanupF) cancelable {
	c.cleanupAction = cleanup
	return c
}

func (c cancelable) call() (interface{}, error) {
	return callBlockingAction(c.blockingAction, c.cleanupAction, c.cancelChannel)
}

func (c cancelable) cancel() {
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
