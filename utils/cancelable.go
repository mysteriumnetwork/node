package utils

import "errors"

// BlockingOperation type defines a function which takes no arguments and is expected to return val and error as result
// and is expected to be blocking (i.e. some request to external system)
type BlockingOperation func() (interface{}, error)

// CleanupCallback type defines a function which is called when blocking operation returns but it was Cancelled,
// callback parameters are val and error returned by blocking operation
type CleanupCallback func(val interface{}, err error)

// CancelChannel defines a channel which is used to monitor when blocking action is Cancelled
type CancelChannel chan int

// ErrRequestCancelled error is returned by CancelableAction.Call() method when Cancel() method was called
var ErrRequestCancelled = errors.New("request was cancelled")

// ErrUndefinedRequest error is returned by CancelableAction.Call() by default when BlockingOperation was not defined
// ( CancelableAction.Request(BlockingOperation) was not called )
var ErrUndefinedRequest = errors.New("undefined request called")

// SkipOnError decorator takes callback function with single val as parameter and returns CleanupCallback function, which calls
// given callback only if no error was returned by blocking operation (i.e. good when cleanup is need only when blocking operation succeded but was Cancelled before)
func SkipOnError(callback func(interface{})) CleanupCallback {
	return func(val interface{}, err error) {
		if err == nil {
			callback(val)
		}
	}
}

// CancelableAction structure represents a possibly long running action which can be Cancelled by another thread
// All "builder" methods (i.e. Request, Cleanup) return a new copy CancelableAction and can specify different operations without
// interference. However cancel channel is shared between those copies, as a result - Cancel() can cancel all outstanding operations
// initiated from single CancelableAction
type CancelableAction struct {
	// Cancelled channel is closed when operation was cancelled
	Cancelled     CancelChannel
	requestAction BlockingOperation
	cleanupAction CleanupCallback
	cancelAction  func()
}

func noCleanup(interface{}, error) {

}

func undefinedRequest() (interface{}, error) {
	return nil, ErrUndefinedRequest
}

// NewCancelable function returns new CancelableAction with default request undefined, and "do nothing" cleanup callback
func NewCancelable() CancelableAction {
	channel := make(CancelChannel, 1)
	return CancelableAction{
		Cancelled:     channel,
		requestAction: undefinedRequest,
		cleanupAction: noCleanup,
		cancelAction: CallOnce(func() {
			close(channel)
		}),
	}
}

// Request method takes long running operation function as parameter and returns copy of CancelableAction as a result
func (c CancelableAction) Request(method BlockingOperation) CancelableAction {
	c.requestAction = method
	return c
}

// Cleanup method takes CleanupCallback as parameter and returns a copy of CancelableAction as a result
func (c CancelableAction) Cleanup(cleanup CleanupCallback) CancelableAction {
	c.cleanupAction = cleanup
	return c
}

// Call method initiates blocking operation, waits for operation to be completed or Cancelled and returns operation result as first val, or error as second
func (c CancelableAction) Call() (interface{}, error) {
	return callBlockingAction(c.requestAction, c.cleanupAction, c.Cancelled)
}

// Cancel method forces Call() method to return imediatelly with ErrRequestCancelled
func (c CancelableAction) Cancel() {
	c.cancelAction()
}

type actionResult struct {
	val interface{}
	err error
}

type actionResultChannel chan actionResult

func callBlockingAction(blockingOp BlockingOperation, cleanup CleanupCallback, cancel CancelChannel) (interface{}, error) {

	resultChannel := make(actionResultChannel, 1)

	go func() {
		val, err := blockingOp()
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
		return nil, ErrRequestCancelled
	}
}
