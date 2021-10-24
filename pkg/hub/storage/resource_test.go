package datahubstorage

import (
	"errors"
	"io"
	"testing"

	. "github.com/franela/goblin"
)

func newThreadSafeResourceWithTestData(g *G) (*threadSafeResource, []byte) {
	resource := newThreadSafeResource()
	testData := []byte{0x1, 0x2, 0x3, 0x5, 0x6}

	n, err := resource.Write(testData)

	g.Assert(n).Equal(len(testData), "not all data was written to resource")
	g.Assert(err).IsNil("should not return any error while writing open resource")

	return &resource, testData
}

func TestResource(t *testing.T) {
	g := Goblin(t)

	g.Describe("threadSafeResource", func() {
		g.It("Should correctly write and read data from resource", func() {
			resource, testData := newThreadSafeResourceWithTestData(g)

			result := make([]byte, 5)
			n, err := resource.ReadAt(result, 0)

			g.Assert(n).Equal(len(testData), "not all data was written to result slice")
			g.Assert(err).IsNil("should not return error when reading resource")
			g.Assert(result).Equal(testData, "result data was not copied correctly")
		})

		g.It("ReadAt should return io.ErrNoProgress if offset is out of data range and resource is not closed", func() {
			resource, _ := newThreadSafeResourceWithTestData(g)

			result := make([]byte, 5)
			_, err := resource.ReadAt(result, 6)

			g.Assert(err).Equal(io.ErrNoProgress)
		})

		g.It("ReadAt should return io.EOF if offset is out of data range and resource is closed", func() {
			resource, _ := newThreadSafeResourceWithTestData(g)

			resource.Close(nil)
			result := make([]byte, 5)
			_, err := resource.ReadAt(result, 6)

			g.Assert(err).Equal(io.EOF)
		})

		g.It("ReadAt should return io.EOF if offset is out of data range and resource is closed", func() {
			resource, _ := newThreadSafeResourceWithTestData(g)

			resource.Close(nil)
			result := make([]byte, 5)
			_, err := resource.ReadAt(result, 6)

			g.Assert(err).Equal(io.EOF)
		})

		g.It("Write should return ResourceClosedForWriting when trying to write to closed resource", func() {
			resource, _ := newThreadSafeResourceWithTestData(g)

			resource.Close(nil)
			_, err := resource.Write([]byte{0x7, 0x8, 0x9})

			g.Assert(err).Equal(errResourceClosedForWriting)
		})

		g.It("Close should return errAlreadyClosed when trying to close already closed resource", func() {
			resource, _ := newThreadSafeResourceWithTestData(g)

			resource.Close(nil)
			err := resource.Close(nil)

			g.Assert(err).Equal(errResourceAlreadyClosed)
		})

		g.It("ReadAt should return given error when closed with non-nil error", func() {
			resource, _ := newThreadSafeResourceWithTestData(g)
			testError := errors.New("test error")

			resource.Close(testError)
			_, err := resource.ReadAt([]byte{}, 0)

			g.Assert(err).Equal(testError)
		})
	})
}
