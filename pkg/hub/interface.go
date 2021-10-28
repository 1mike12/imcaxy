package hub

import (
	"context"
	"errors"
	"io"
)

type (
	DataStreamInput interface {
		io.Writer
		io.ReaderFrom

		Close(errorToForward error) error
	}

	DataStreamOutput interface {
		io.ReadSeekCloser
		io.ReaderAt
		io.WriterTo
	}

	DataHub interface {
		StartMonitors(ctx context.Context)
		CreateStream(streamID string) (DataStreamInput, error)
		GetStreamOutput(streamID string) (DataStreamOutput, error)

		// If DataStreamInput is nil, it means that stream is already writing in other
		// goroutine, so we can just read the data that is sent through stream without
		// need to double the work and manually download and write it.
		GetOrCreateStream(streamID string) (DataStreamOutput, DataStreamInput, error)
	}
)

var (
	ErrOffsetOutOfRange   = errors.New("offset out of range")
	ErrSeekEndUnsupported = errors.New("io.SeekEnd whence is unsupported")
)
