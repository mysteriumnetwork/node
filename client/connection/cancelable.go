package connection

import "errors"

type blockingF func() (interface{}, error)

type cleanupF func(interface{}, error)

func noCleanup(interface{}, error) {

}

type cancelChannel chan int

var errActionCancelled = errors.New("action was cancelled")

type functionResult struct {
	val interface{}
	err error
}

type functionResultChannel chan functionResult

func cancelableActionWithCleanup(method blockingF, cleanup cleanupF, cancel cancelChannel) (interface{}, error) {

	resultChannel := make(functionResultChannel, 1)

	go func() {
		val, err := method()
		resultChannel <- functionResult{val, err}
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

func cancelableAction(method blockingF, cancel cancelChannel) (interface{}, error) {
	return cancelableActionWithCleanup(method, noCleanup, cancel)
}
