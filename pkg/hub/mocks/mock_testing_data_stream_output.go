package mock_hub

import (
	"time"

	"github.com/franela/goblin"
	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type MockTestingDataStreamOutput struct {
	hub.DataStreamOutput

	g      *goblin.G
	reader *mockTestingDataStreamOutputReader
}

var _ hub.DataStreamOutput = (*MockTestingDataStreamOutput)(nil)

func NewMockTestingDataStreamOutput(
	g *goblin.G,
	responses [][]byte,
	lastReadError error,
	closeError error,
) MockTestingDataStreamOutput {
	reader := mockTestingDataStreamOutputReader{
		responses,
		lastReadError,
		closeError,
		make(chan struct{}, 1),
	}

	dataStreamOutput := hub.NewDataStreamOutput(&reader)

	return MockTestingDataStreamOutput{
		&dataStreamOutput,
		g, &reader,
	}
}

func (stream *MockTestingDataStreamOutput) Wait() {
	select {
	case <-time.After(time.Second):
		stream.g.Errorf("MockTestingDataStreamOutput Wait deadline exceeded")
	case <-stream.reader.finisher:
		return
	}
}

type mockTestingDataStreamOutputReader struct {
	responses     [][]byte
	lastReadError error
	closeError    error
	finisher      chan struct{}
}

func (reader *mockTestingDataStreamOutputReader) ReadAt(p []byte, off int64) (n int, err error) {
	dataSegment := reader.getResponseSegmentAtOffset(off)
	if dataSegment == nil {
		return 0, reader.lastReadError
	}

	return copy(p, dataSegment), nil
}

func (reader *mockTestingDataStreamOutputReader) Close() error {
	reader.finisher <- struct{}{}
	return reader.closeError
}

func (reader *mockTestingDataStreamOutputReader) getResponseSegmentAtOffset(off int64) []byte {
	totalPos := int64(0)

	for _, response := range reader.responses {
		if totalPos >= off {
			return response
		}

		totalPos += int64(len(response))
	}

	return nil
}
