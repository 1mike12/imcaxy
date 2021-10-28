package hub

import (
	"context"
	"sync"

	datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage"
)

type dataHub struct {
	storage datahubstorage.StorageAdapter
	lock    sync.RWMutex
}

var _ DataHub = (*dataHub)(nil)

func NewDataHub(storage datahubstorage.StorageAdapter) DataHub {
	return &dataHub{storage, sync.RWMutex{}}
}

func (hub *dataHub) StartMonitors(ctx context.Context) {
	hub.storage.StartMonitors(ctx)
}

func (hub *dataHub) CreateStream(streamID string) (DataStreamInput, error) {
	hub.lock.Lock()
	defer hub.lock.Unlock()

	return hub.createStream(streamID)
}

func (hub *dataHub) GetStreamOutput(streamID string) (DataStreamOutput, error) {
	hub.lock.RLock()
	defer hub.lock.RUnlock()

	return hub.getStreamOutput(streamID)
}

func (hub *dataHub) GetOrCreateStream(streamID string) (output DataStreamOutput, input DataStreamInput, err error) {
	hub.lock.Lock()
	defer hub.lock.Unlock()

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
