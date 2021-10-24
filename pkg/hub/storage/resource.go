package datahubstorage

import (
	"errors"
	"io"
	"sync"
)

type threadSafeResource struct {
	data           []byte
	closed         bool  // all of the contents of resource is already written
	errorToForward error // error that ocurred while reading resource data
	lock           sync.RWMutex
}

func newThreadSafeResource() threadSafeResource {
	return threadSafeResource{
		make([]byte, 0),
		false,
		nil,
		sync.RWMutex{},
	}
}

func (res *threadSafeResource) ReadAt(p []byte, off int64) (n int, err error) {
	res.lock.RLock()
	defer res.lock.RUnlock()

	if res.errorToForward != nil {
		return 0, res.errorToForward
	}

	if off >= int64(len(res.data)) {
		if res.closed {
			err = io.EOF
			return
		}

		err = io.ErrNoProgress
		return
	}

	n = copy(p, res.data[off:])
	return
}

func (res *threadSafeResource) Write(p []byte) (n int, err error) {
	res.lock.Lock()
	defer res.lock.Unlock()

	if res.closed {
		err = errResourceClosedForWriting
		return
	}

	res.data = append(res.data, p...)
	n = len(p)
	return
}

func (res *threadSafeResource) Close(errorToForward error) error {
	res.lock.Lock()
	defer res.lock.Unlock()

	if res.closed {
		return errResourceAlreadyClosed
	}

	res.closed = true
	res.errorToForward = errorToForward
	return nil
}

var (
	errResourceAlreadyClosed    = errors.New("resource already closed")
	errResourceClosedForWriting = errors.New("resource closed for writing")
)
