package hub

import (
	"bytes"
	"errors"
	"io"
	"testing"

	. "github.com/franela/goblin"
	"github.com/golang/mock/gomock"
	mock_datahubstorage "github.com/thebartekbanach/imcaxy/pkg/hub/storage/mocks"
)

type readResult struct {
	n   int
	err error
}

func TestDataStreamOutput(t *testing.T) {
	g := Goblin(t)

	g.Describe("DataStreamOutput", func() {
		g.It("Should correctly read all data from storage", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)
			mockReader.EXPECT().ReadAt(gomock.Any(), int64(0)).Return(3, nil).Times(1)
			mockReader.EXPECT().ReadAt(gomock.Any(), int64(3)).Return(3, io.EOF).Times(1)
			expectedResults := []readResult{
				{3, nil},
				{3, io.EOF},
			}

			stream := NewDataStreamOutput(mockReader)
			for _, expected := range expectedResults {
				data := make([]byte, 3)
				n, err := stream.Read(data)

				g.Assert(n).Equal(expected.n)
				g.Assert(err).Equal(expected.err)
			}
		})

		g.It("Should correctly seek internal offset from begin of resource", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)
			mockReader.EXPECT().ReadAt(gomock.Any(), int64(3)).Return(3, io.EOF).Times(1)

			stream := NewDataStreamOutput(mockReader)
			stream.Seek(3, io.SeekStart)
			data := make([]byte, 3)
			stream.Read(data)
		})

		g.It("Should correctly seek internal offset from current position", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)
			mockReader.EXPECT().ReadAt(gomock.Any(), int64(3)).Return(3, io.EOF).Times(1)

			stream := NewDataStreamOutput(mockReader)
			stream.Seek(1, io.SeekStart)
			stream.Seek(2, io.SeekCurrent)
			data := make([]byte, 3)
			stream.Read(data)
		})

		g.It("Should return error when if trying to seek internal offset from end of resource", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)

			stream := NewDataStreamOutput(mockReader)
			_, err := stream.Seek(3, io.SeekEnd)

			g.Assert(err).Equal(ErrSeekEndUnsupported)
		})

		g.It("Should return error if trying to seek internal offset to negative value", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)

			stream := NewDataStreamOutput(mockReader)
			_, err := stream.Seek(-1, io.SeekStart)

			g.Assert(err).Equal(ErrOffsetOutOfRange)
		})

		g.It("Should close stream reader correctly", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)
			mockReader.EXPECT().Close().Times(1)

			stream := NewDataStreamOutput(mockReader)
			stream.Close()
		})

		g.It("Should return error when trying to close already closed stream", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)
			mockReader.EXPECT().Close().Return(nil)

			stream := NewDataStreamOutput(mockReader)
			stream.Close()
			err := stream.Close()

			g.Assert(err).Equal(ErrStreamAlreadyClosed)
		})

		g.It("Should return error when trying to Read from closed stream", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)
			mockReader.EXPECT().Close().Return(nil)

			stream := NewDataStreamOutput(mockReader)
			stream.Close()
			_, err := stream.Read([]byte{})

			g.Assert(err).Equal(ErrStreamClosedForReading)
		})

		g.It("Should return error when trying to ReadAt from closed stream", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)
			mockReader.EXPECT().Close().Return(nil)

			stream := NewDataStreamOutput(mockReader)
			stream.Close()
			_, err := stream.ReadAt([]byte{}, 0)

			g.Assert(err).Equal(ErrStreamClosedForReading)
		})

		g.It("Should return error when trying to Seek on closed stream", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)
			mockReader.EXPECT().Close().Return(nil)

			stream := NewDataStreamOutput(mockReader)
			stream.Close()
			_, err := stream.Seek(0, io.SeekStart)

			g.Assert(err).Equal(ErrStreamClosedForReading)
		})

		g.It("Should forward stream reader close error", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			closeError := errors.New("some close error")
			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)
			mockReader.EXPECT().Close().Return(closeError).Times(1)

			stream := NewDataStreamOutput(mockReader)
			err := stream.Close()

			g.Assert(err).Equal(closeError)
		})

		g.It("Should ReadAt correctly", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)
			mockReader.EXPECT().ReadAt(gomock.Any(), int64(3)).Return(3, nil).Times(1)

			stream := NewDataStreamOutput(mockReader)
			data := make([]byte, 3)
			stream.ReadAt(data, 3)
		})

		g.It("Should write all data using WriteTo method", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			data := []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6}
			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)

			mockReader.EXPECT().ReadAt(gomock.Any(), int64(0)).DoAndReturn(func(p []byte, off int64) (int, error) {
				n := copy(p, data[off:3])
				return n, nil
			}).Times(1)

			mockReader.EXPECT().ReadAt(gomock.Any(), int64(3)).DoAndReturn(func(p []byte, off int64) (int, error) {
				n := copy(p, data[off:off+3])
				return n, io.EOF
			}).Times(1)

			stream := NewDataStreamOutput(mockReader)
			var result bytes.Buffer
			n, err := stream.WriteTo(&result)

			g.Assert(result.Bytes()).Equal(data)
			g.Assert(n).Equal(int64(6))
			g.Assert(err).Equal(io.EOF)
		})

		g.It("Should return error when trying to use WriteTo on closed stream", func() {
			mockCtrl := gomock.NewController(g)
			defer mockCtrl.Finish()

			mockReader := mock_datahubstorage.NewMockStreamReader(mockCtrl)
			mockReader.EXPECT().Close().Return(nil)

			stream := NewDataStreamOutput(mockReader)
			stream.Close()
			_, err := stream.Read([]byte{})

			g.Assert(err).Equal(ErrStreamClosedForReading)
		})
	})
}
