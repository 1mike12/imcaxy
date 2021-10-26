package datahubstorage

import (
	"context"
	"io"
)

type (
	StreamReader interface {
		io.ReaderAt
		io.Closer
	}

	Writer interface {
		Create(streamID string) error
		Write(streamID string, p []byte) (n int, err error)
		Close(streamID string, errorToForward error) error
	}

	Reader interface {
		GetStreamReader(streamID string) (StreamReader, error)
	}

	StorageAdapter interface {
		Writer
		Reader

		// start background tasks
		StartMonitors(ctx context.Context)
	}
)
