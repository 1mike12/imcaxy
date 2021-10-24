package datahubstorage

import (
	"errors"
	"sync"
)

type readersList struct {
	lock           sync.Mutex
	readers        map[string]int
	streamReleased chan string
}

func newReadersList() readersList {
	return readersList{
		sync.Mutex{},
		make(map[string]int),
		make(chan string, 128),
	}
}

func (list *readersList) Created(streamID string) {
	list.lock.Lock()
	defer list.lock.Unlock()

	if _, exists := list.readers[streamID]; !exists {
		list.readers[streamID] = 0
	}

	list.readers[streamID]++
}

func (list *readersList) Closed(streamID string) error {
	list.lock.Lock()
	defer list.lock.Unlock()

	readers, exists := list.readers[streamID]
	if !exists {
		return errReaderDoesNotExist
	}

	updatedReadersNumber := readers - 1
	list.readers[streamID] = updatedReadersNumber

	if updatedReadersNumber < 1 {
		delete(list.readers, streamID)
		list.streamReleased <- streamID
	}

	return nil
}

func (list *readersList) OnRelease() <-chan string {
	return list.streamReleased
}

var errReaderDoesNotExist = errors.New("reader does not exist")
