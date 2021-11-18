package datahubstorage

import (
	"context"
	"errors"
	"io"
)

type Storage struct {
	readersList     readersList
	notificationHub notificationHub
	resourceList    resourceList
}

var _ StorageAdapter = (*Storage)(nil)

func NewStorage() StorageAdapter {
	return &Storage{
		newReadersList(),
		newNotificationHub(),
		newResourceList(),
	}
}

func (storage *Storage) StartMonitors(ctx context.Context) {
	go storage.notificationHub.StartMonitor(ctx)
	go storage.startDisposer(ctx)
}

func (storage *Storage) Create(streamID string) error {
	err := storage.resourceList.Create(streamID)
	if err != nil {
		return ErrStreamAlreadyExists
	}

	err = <-storage.notificationHub.RegisterTopic(streamID)
	if err != nil {
		storage.resourceList.Dispose(streamID)
		return ErrStreamAlreadyExists
	}

	storage.readersList.Created(streamID)
	return nil
}

func (storage *Storage) Write(streamID string, p []byte) (n int, err error) {
	n, err = storage.resourceList.Write(streamID, p)
	switch err {
	case errResourceClosedForWriting:
		return 0, ErrStreamClosedForWriting

	case errUnknownResource:
		return 0, ErrUnknownStream

	case nil:
		<-storage.notificationHub.SendNotification(streamID)
		return n, nil

	default:
		return n, err
	}

}

func (storage *Storage) Close(streamID string, errorToForward error) error {
	err := storage.resourceList.Close(streamID, errorToForward)

	switch err {
	case errResourceAlreadyClosed:
		return ErrStreamAlreadyClosed

	case errUnknownResource:
		return ErrUnknownStream

	default:
		<-storage.notificationHub.CloseTopic(streamID, errorToForward)
		storage.readersList.Closed(streamID)
		return nil
	}
}

func (storage *Storage) GetStreamReader(streamID string) (StreamReader, error) {
	// How it is all synchronized?
	// First we are adding one reader to our readers list,
	// it has its own mutex inside.
	// Then we are checking if resource exists.
	// In case when it does not exist, we are removing
	// our reader from readersList, because we have not created it really.
	// Otherwise we are just creating reader and returning it.

	storage.readersList.Created(streamID)

	if !storage.resourceList.Exists(streamID) {
		storage.readersList.Closed(streamID)
		return nil, ErrUnknownStream
	}

	reader := streamReader{
		streamID,
		storage,
	}

	return &reader, nil
}

func (storage *Storage) readAt(streamID string, p []byte, off int64) (n int, err error) {
	n, err = storage.resourceList.ReadAt(streamID, p, off)
	if err == io.ErrNoProgress {
		notification := <-storage.notificationHub.OnNotify(streamID)
		if notification.err != nil && notification.err != errTopicNotFound {
			return 0, notification.err
		}

		return storage.readAt(streamID, p, off)
	}

	return
}

func (storage *Storage) readerClosed(streamID string) error {
	return storage.readersList.Closed(streamID)
}

func (storage *Storage) startDisposer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case releasedStreamID := <-storage.readersList.OnRelease():
			storage.resourceList.Dispose(releasedStreamID)
		}
	}
}

type streamReader struct {
	streamID string
	storage  *Storage
}

var _ StreamReader = (*streamReader)(nil)

func (reader *streamReader) ReadAt(p []byte, off int64) (n int, err error) {
	return reader.storage.readAt(reader.streamID, p, off)
}

func (reader *streamReader) Close() error {
	return reader.storage.readerClosed(reader.streamID)
}

var (
	ErrUnknownStream          = errors.New("unknown stream")
	ErrStreamClosedForWriting = errors.New("stream closed for writing")
	ErrStreamAlreadyExists    = errors.New("already exists")
	ErrStreamAlreadyClosed    = errors.New("already closed")
)
