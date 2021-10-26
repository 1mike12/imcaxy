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
	}
)

var (
	ErrOffsetOutOfRange   = errors.New("offset out of range")
	ErrSeekEndUnsupported = errors.New("io.SeekEnd whence is unsupported")
)
