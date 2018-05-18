/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package utils

import "errors"

// BlockingRequest type defines a function which takes no arguments and is expected to return val and error as result
// and is expected to be blocking (i.e. some request to external system)
type BlockingRequest func() (interface{}, error)

// CleanupCallback type defines a function which is called when blocking operation returns but it was Cancelled,
// callback parameters are val and error returned by blocking operation
type CleanupCallback func(val interface{}, err error)

// CancelChannel defines a channel which is used to monitor when blocking action is Cancelled
type CancelChannel chan int

// ErrRequestCancelled error is returned by CancelableRequest.Call() method when Cancel() method was called
var ErrRequestCancelled = errors.New("request was cancelled")

// InvokeOnSuccess decorator takes callback function with single val as parameter and returns CleanupCallback function, which calls
// given callback only if no error was returned by blocking request (i.e. good when cleanup is need only when blocking request succeded but was canceled before)
func InvokeOnSuccess(callback func(interface{})) CleanupCallback {
	return func(val interface{}, err error) {
		if err == nil {
			callback(val)
		}
	}
}

// Cancelable structure represents object which can be cancelable and holds channel which is closed on cancel
type Cancelable struct {
	// Cancelled channel is closed when operation was cancelled
	Cancelled    CancelChannel
	cancelAction func()
}

// NewCancelable creates new cancelable object which can then be used to created cancelable requests
func NewCancelable() Cancelable {
	channel := make(CancelChannel, 1)
	return Cancelable{
		Cancelled: channel,
		cancelAction: CallOnce(func() {
			close(channel)
		}),
	}
}

// Cancel method signal all created requests to exit immediatelly
func (c Cancelable) Cancel() {
	c.cancelAction()
}

// No op cleanup callback by default
func noCleanup(interface{}, error) {

}

// CancelableRequest represents object which calls blocking action and can be cancelled - i.e. Call returns early with posibility cleanup blocking action
// results if needed
type CancelableRequest struct {
	canceled CancelChannel
	request  BlockingRequest
	cleanup  CleanupCallback
}

// NewRequest method returns new CancelableRequest which can be cancelled by calling cancelables Cancel method
func (c Cancelable) NewRequest(request BlockingRequest) *CancelableRequest {
	return &CancelableRequest{
		canceled: c.Cancelled,
		request:  request,
		cleanup:  noCleanup,
	}
}

// Cleanup method setups CleanupCallback in case result of blocking actions needs to be cleaned up
func (c *CancelableRequest) Cleanup(cleanup CleanupCallback) *CancelableRequest {
	c.cleanup = cleanup
	return c
}

// Call method initiates blocking request, waits for requests to be completed or canceled and returns request result as first val, or error as second
func (c *CancelableRequest) Call() (interface{}, error) {
	return callBlockingRequest(c.request, c.cleanup, c.canceled)
}

type requestResult struct {
	val interface{}
	err error
}

type requestResultChannel chan requestResult

func callBlockingRequest(blockingOp BlockingRequest, cleanup CleanupCallback, canceled CancelChannel) (interface{}, error) {

	resultChannel := make(requestResultChannel, 1)

	go func() {
		val, err := blockingOp()
		resultChannel <- requestResult{val, err}
	}()

	select {
	case res := <-resultChannel:
		return res.val, res.err
	case <-canceled:
		go func() {
			res := <-resultChannel
			cleanup(res.val, res.err)
		}()
		return nil, ErrRequestCancelled
	}
}
