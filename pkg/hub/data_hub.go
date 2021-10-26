package hub

import (
	"context"

	datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage"
)

type dataHub struct {
	storage datahubstorage.StorageAdapter
}

var _ DataHub = (*dataHub)(nil)

func NewDataHub(storage datahubstorage.StorageAdapter) DataHub {
	return &dataHub{storage}
}

func (hub *dataHub) StartMonitors(ctx context.Context) {
	hub.storage.StartMonitors(ctx)
}

func (hub *dataHub) CreateStream(streamID string) (DataStreamInput, error) {
	if err := hub.storage.Create(streamID); err != nil {
		return nil, err
	}

	input := newDataStreamInput(streamID, hub.storage)
	return &input, nil
}

func (hub *dataHub) GetStreamOutput(streamID string) (DataStreamOutput, error) {
	streamReader, err := hub.storage.GetStreamReader(streamID)
	if err != nil {
		return nil, err
	}

	streamOutput := newDataStreamOutput(streamReader)
	return &streamOutput, nil
}
