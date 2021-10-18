package hub

import (
	"io"
)

type (
	DataStreamInput interface {
		io.WriteCloser
		io.ReaderFrom
	}

	DataStreamOutput interface {
		io.ReadSeekCloser
		io.ReaderAt
		io.WriterTo

		IsAvailable() bool
		ReadDone() bool
	}

	DataHub interface {
		CreateStream(streamID string) (DataStreamInput, error)
		GetStreamOutput(streamID string) (DataStreamOutput, error)
	}
)
