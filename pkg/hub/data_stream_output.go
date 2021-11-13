package hub

import (
	"io"
	"sync"

	datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage"
)

type dataStreamOutput struct {
	reader datahubstorage.StreamReader
	pos    int64
	closed bool
	lock   sync.Mutex
}

var _ DataStreamOutput = (*dataStreamOutput)(nil)

func NewDataStreamOutput(reader datahubstorage.StreamReader) dataStreamOutput {
	return dataStreamOutput{
		reader,
		0,
		false,
		sync.Mutex{},
	}
}

func (stream *dataStreamOutput) Read(p []byte) (n int, err error) {
	stream.lock.Lock()
	defer stream.lock.Unlock()

	if stream.closed {
		err = ErrStreamClosedForReading
		return
	}

	n, err = stream.reader.ReadAt(p, stream.pos)
	if err != nil {
		return
	}

	stream.pos += int64(n)
	return
}

func (stream *dataStreamOutput) Seek(offset int64, whence int) (n int64, err error) {
	stream.lock.Lock()
	defer stream.lock.Unlock()

	n = stream.pos

	if stream.closed {
		err = ErrStreamClosedForReading
		return
	}

	if stream.pos+offset < 0 {
		err = ErrOffsetOutOfRange
		return
	}

	if whence == io.SeekEnd {
		err = ErrSeekEndUnsupported
		return
	}

	if whence == io.SeekStart {
		stream.pos = offset
		n = stream.pos
		return
	}

	// whence == io.SeekCurrent
	stream.pos += offset
	n = stream.pos
	return
}

func (stream *dataStreamOutput) Close() error {
	stream.lock.Lock()
	defer stream.lock.Unlock()

	if stream.closed {
		return ErrStreamAlreadyClosed
	}

	stream.closed = true
	return stream.reader.Close()
}

func (stream *dataStreamOutput) ReadAt(p []byte, off int64) (n int, err error) {
	stream.lock.Lock()
	defer stream.lock.Unlock()

	if stream.closed {
		err = ErrStreamClosedForReading
		return
	}

	n, err = stream.reader.ReadAt(p, off)
	return
}

func (stream *dataStreamOutput) WriteTo(w io.Writer) (n int64, err error) {
	stream.lock.Lock()
	defer stream.lock.Unlock()

	pos := int64(0)
	eof := false
	for !eof {
		data := make([]byte, 256)

		numOfReadBytes, err := stream.reader.ReadAt(data, pos)
		switch err {
		case io.EOF:
			eof = true
		case nil:
		default:
			return pos, err
		}

		dataToWrite := data[:numOfReadBytes]
		numOfWrittenBytes, err := w.Write(dataToWrite)
		pos += int64(numOfWrittenBytes)

		if err != nil {
			return pos, err
		}
	}

	return pos, io.EOF
}
