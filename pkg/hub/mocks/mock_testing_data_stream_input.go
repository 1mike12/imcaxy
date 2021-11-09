package mock_hub

import (
	"bytes"
	"io"
	"time"

	"github.com/thebartekbanach/imcaxy/pkg/hub"
)

type MockTestingDataStreamInput struct {
	t T

	expectedResponses [][]byte
	writeLastResponse error
	closeResponse     error

	DataSegments   [][]byte
	ForwardedError error

	finisher chan struct{}
}

var _ hub.DataStreamInput = (*MockTestingDataStreamInput)(nil)

func NewMockTestingDataStreamInput(t T, expectedResponses [][]byte, writeLastResponse error, closeResponse error) MockTestingDataStreamInput {
	return MockTestingDataStreamInput{
		t,

		expectedResponses,
		writeLastResponse,
		closeResponse,

		make([][]byte, 0),
		nil,

		make(chan struct{}, 1),
	}
}

func (stream *MockTestingDataStreamInput) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	stream.DataSegments = append(stream.DataSegments, p)

	if stream.expectedResponses != nil {
		responseIndex := len(stream.DataSegments) - 1
		expectedResponse := stream.expectedResponses[responseIndex]

		if !bytes.Equal(expectedResponse, p) {
			stream.t.Errorf(
				"DataStreamInput Write was called with wrong set of data to write (index: %v), expected %v, got %v",
				responseIndex, expectedResponse, p,
			)
		}

		if len(stream.DataSegments) == len(stream.expectedResponses) {
			return len(p), stream.writeLastResponse
		}
	}

	return len(p), nil
}

func (stream *MockTestingDataStreamInput) Close(errorToForward error) error {
	stream.ForwardedError = errorToForward
	if len(stream.finisher) == 0 {
		stream.finisher <- struct{}{}
	}
	return stream.closeResponse
}

func (stream *MockTestingDataStreamInput) ReadFrom(r io.Reader) (int64, error) {
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

func (stream *MockTestingDataStreamInput) Wait() {
	select {
	case <-time.After(time.Second):
		stream.t.Errorf("MockTestingDataStreamInput Wait deadline exceeded")
	case <-stream.finisher:
		return
	}
}

func (stream *MockTestingDataStreamInput) SafelyGetDataSegment(segmentIndex int) []byte {
	if segmentIndex >= len(stream.DataSegments) {
		return nil
	}

	return stream.DataSegments[segmentIndex]
}

func (stream *MockTestingDataStreamInput) GetWholeResponse() []byte {
	return bytes.Join(stream.DataSegments, []byte(""))
}
