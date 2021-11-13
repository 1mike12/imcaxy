package hub

import (
	"io"

	datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage"
)

type dataStreamInput struct {
	streamID string
	storage  datahubstorage.Writer
	closed   bool
}

var _ DataStreamInput = (*dataStreamInput)(nil)

func newDataStreamInput(streamID string, storage datahubstorage.Writer) dataStreamInput {
	return dataStreamInput{
		streamID,
		storage,
		false,
	}
}

func (stream *dataStreamInput) Write(p []byte) (n int, err error) {
	if stream.closed {
		return 0, ErrStreamClosedForWriting
	}

	return stream.storage.Write(stream.streamID, p)
}

func (stream *dataStreamInput) Close(errorToForward error) error {
	if stream.closed {
		return ErrStreamAlreadyClosed
	}
	stream.closed = true

	if errorToForward == io.EOF {
		errorToForward = nil
	}

	return stream.storage.Close(stream.streamID, errorToForward)
}

func (stream *dataStreamInput) ReadFrom(r io.Reader) (int64, error) {
	if stream.closed {
		return 0, ErrStreamClosedForWriting
	}

	fullSize := int64(0)
	for {
		data := make([]byte, 256)

		readSize, readErr := r.Read(data)
		fullSize += int64(readSize)

		if readErr != nil && readErr != io.EOF {
			return fullSize, readErr
		}

		if _, writeErr := stream.Write(data[:readSize]); writeErr != nil {
			return fullSize, writeErr
		}

		if readErr == io.EOF {
			return fullSize, io.EOF
		}
	}
}
