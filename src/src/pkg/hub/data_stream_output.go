package hub

import (
	"io"
)

type dataStreamOutput struct {
}

var _ DataStreamOutput = (*dataStreamOutput)(nil)

func (stream *dataStreamOutput) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (stream *dataStreamOutput) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (stream *dataStreamOutput) Close() error {
	return nil
}

func (stream *dataStreamOutput) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, nil
}

func (stream *dataStreamOutput) WriteTo(w io.Writer) (n int64, err error) {
	return 0, nil
}

func (stream *dataStreamOutput) IsAvailable() bool {
	return false
}

func (stream *dataStreamOutput) ReadDone() bool {
	return false
}
