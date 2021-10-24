package datahubstorage

import (
	"errors"
	"testing"

	. "github.com/franela/goblin"
)

func TestResourceList(t *testing.T) {
	g := Goblin(t)

	g.Describe("resourceList", func() {
		g.It("should create resource without error", func() {
			list := newResourceList()

			err := list.Create("test")

			g.Assert(err).IsNil()
		})

		g.It("Create should return errResourceAlreadyExists error if resource already exists", func() {
			list := newResourceList()

			list.Create("test")
			err := list.Create("test")

			g.Assert(err).Equal(errResourceAlreadyExists)
		})

		g.It("should close given resource without error", func() {
			list := newResourceList()

			list.Create("test")
			err := list.Close("test", nil)

			g.Assert(err).IsNil()

			_, err = list.Write("test", []byte{0x0})

			g.Assert(err).Equal(errResourceClosedForWriting)
		})

		g.It("Close should return error errUnknownResource if resource is not available", func() {
			list := newResourceList()

			err := list.Close("unknown", nil)

			g.Assert(err).Equal(errUnknownResource)
		})

		g.It("should dispose given resource without error", func() {
			list := newResourceList()

			list.Create("test")
			err := list.Dispose("test")

			g.Assert(err).IsNil()

			_, err = list.Write("test", []byte{0x0})

			g.Assert(err).Equal(errUnknownResource)
		})

		g.It("Dispose should return errUnknownResource if resource is not available", func() {
			list := newResourceList()

			err := list.Dispose("test")

			g.Assert(err).Equal(errUnknownResource)
		})

		g.It("Write should return errUnknownResource if resource is not available", func() {
			list := newResourceList()

			_, err := list.Write("test", []byte{0x0})

			g.Assert(err).Equal(errUnknownResource)
		})

		g.It("ReadAt should return errUnknownResource if resource is not available", func() {
			list := newResourceList()

			data := make([]byte, 5)
			_, err := list.ReadAt("test", data, 0)

			g.Assert(err).Equal(errUnknownResource)
		})

		g.It("ReadAt should return given error when closed with non-nil error", func() {
			list := newResourceList()
			testError := errors.New("test error")

			list.Create("test")
			list.Write("test", []byte{0x0})
			list.Close("test", testError)
			_, err := list.ReadAt("test", []byte{}, 0)

			g.Assert(err).Equal(testError)
		})
	})
}
