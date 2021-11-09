package mock_hub

import (
	"bytes"
	"io"
	"time"

	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type MockTestingDataStreamOutput struct {
	hub.DataStreamOutput

	t      T
	reader *mockTestingDataStreamOutputReader
}

var _ hub.DataStreamOutput = (*MockTestingDataStreamOutput)(nil)

type T interface {
	Errorf(format string, args ...interface{})
}

func NewMockTestingDataStreamOutput(
	t T,
	responses [][]byte,
	lastReadError error,
	closeError error,
) MockTestingDataStreamOutput {
	reader := mockTestingDataStreamOutputReader{
		responses,
		lastReadError,
		closeError,
		make(chan struct{}, 1),
		false,
	}

	dataStreamOutput := hub.NewDataStreamOutput(&reader)

	return MockTestingDataStreamOutput{
		&dataStreamOutput,
		t, &reader,
	}
}

func NewMockTestingDataStreamOutputUsingSingleChunkOfData(
	t T,
	data []byte,
	lastReadError error,
	closeError error,
) MockTestingDataStreamOutput {
	reader := mockTestingDataStreamOutputReader{
		[][]byte{data},
		lastReadError,
		closeError,
		make(chan struct{}, 1),
		true,
	}

	dataStreamOutput := hub.NewDataStreamOutput(&reader)

	return MockTestingDataStreamOutput{
		&dataStreamOutput,
		t, &reader,
	}
}

func (stream *MockTestingDataStreamOutput) Wait() {
	select {
	case <-time.After(time.Second):
		stream.t.Errorf("MockTestingDataStreamOutput Wait deadline exceeded")
	case <-stream.reader.finisher:
		return
	}
}

type mockTestingDataStreamOutputReader struct {
	responses              [][]byte
	lastReadError          error
	closeError             error
	finisher               chan struct{}
	usingSingleChunkOfData bool
}

func (reader *mockTestingDataStreamOutputReader) ReadAt(p []byte, off int64) (n int, err error) {
	if reader.usingSingleChunkOfData {
		n, err = bytes.NewReader(reader.responses[0]).ReadAt(p, off)
		return
	}

	dataSegment := reader.getResponseSegmentAtOffset(off)
	if dataSegment == nil {
		if reader.lastReadError != nil {
			return 0, reader.lastReadError
		}

		return 0, io.EOF
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
