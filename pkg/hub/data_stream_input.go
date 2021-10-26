package hub

import (
	"io"

	datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage"
)

type dataStreamInput struct {
	streamID string
	storage  datahubstorage.Writer
}

var _ DataStreamInput = (*dataStreamInput)(nil)

func newDataStreamInput(streamID string, storage datahubstorage.Writer) dataStreamInput {
	return dataStreamInput{
		streamID,
		storage,
	}
}

func (stream *dataStreamInput) Write(p []byte) (n int, err error) {
	return stream.storage.Write(stream.streamID, p)
}

func (stream *dataStreamInput) Close(errorToForward error) error {
	return stream.storage.Close(stream.streamID, errorToForward)
}

func (stream *dataStreamInput) ReadFrom(r io.Reader) (int64, error) {
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
