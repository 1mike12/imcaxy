package hub

import (
	"context"
	"errors"
	"sync"

	datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage"
)

type dataHub struct {
	storage         datahubstorage.StorageAdapter
	lock            sync.RWMutex
	monitorsStarted bool
}

var _ DataHub = (*dataHub)(nil)

func NewDataHub(storage datahubstorage.StorageAdapter) DataHub {
	return &dataHub{storage, sync.RWMutex{}, false}
}

func (hub *dataHub) StartMonitors(ctx context.Context) {
	hub.lock.Lock()
	hub.monitorsStarted = true
	hub.lock.Unlock()

	go hub.storage.StartMonitors(ctx)
}

func (hub *dataHub) CreateStream(streamID string) (DataStreamInput, error) {
	hub.lock.Lock()
	defer hub.lock.Unlock()

	hub.panicIfMonitorsNotStarted()

	return hub.createStream(streamID)
}

func (hub *dataHub) GetStreamOutput(streamID string) (DataStreamOutput, error) {
	hub.lock.RLock()
	defer hub.lock.RUnlock()

	hub.panicIfMonitorsNotStarted()

	return hub.getStreamOutput(streamID)
}

func (hub *dataHub) GetOrCreateStream(streamID string) (output DataStreamOutput, input DataStreamInput, err error) {
	hub.lock.Lock()
	defer hub.lock.Unlock()

	hub.panicIfMonitorsNotStarted()

	input, _ = hub.createStream(streamID)
	output, err = hub.getStreamOutput(streamID)
	if err != nil && input != nil {
		input.Close(err)
	}

	return
}

func (hub *dataHub) createStream(streamID string) (DataStreamInput, error) {
	if err := hub.storage.Create(streamID); err != nil {
		return nil, err
	}

	input := newDataStreamInput(streamID, hub.storage)
	return &input, nil
}

func (hub *dataHub) getStreamOutput(streamID string) (DataStreamOutput, error) {
	streamReader, err := hub.storage.GetStreamReader(streamID)
	if err != nil {
		return nil, err
	}

	streamOutput := NewDataStreamOutput(streamReader)
	return &streamOutput, nil
}

func (hub *dataHub) panicIfMonitorsNotStarted() {
	if !hub.monitorsStarted {
		panic("DataHub monitors not started")
	}
}

var (
	ErrStreamAlreadyClosed    = errors.New("stream already closed")
	ErrStreamClosedForReading = errors.New("stream closed for reading")
	ErrStreamClosedForWriting = errors.New("stream closed for writing")
)
