package hub

import "io"

type dataStreamInput struct {
}

var _ DataStreamInput = (*dataStreamInput)(nil)

func (stream *dataStreamInput) Write(p []byte) (n int, err error) {
	return 0, nil
}

func (stream *dataStreamInput) Close() error {
	return nil
}

func (stream *dataStreamInput) ReadFrom(r io.Reader) (n int64, err error) {
	return 0, nil
}
