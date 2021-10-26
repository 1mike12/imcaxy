package datahubstorage

import (
	"testing"

	. "github.com/franela/goblin"
)

func TestReadersList(t *testing.T) {
	g := Goblin(t)

	g.Describe("readersList", func() {
		g.It("Should notify about stream release", func() {
			list := newReadersList()

			list.Created("test")
			go list.Closed("test")

			if releasedStreamName := <-list.OnRelease(); releasedStreamName != "test" {
				g.Errorf("released stream name is not correct")
			}
		})

		g.It("Should notify about stream release when multiple streams registered", func() {
			list := newReadersList()

			list.Created("test1")
			list.Created("test2")
			go list.Closed("test1")

			if releasedStreamName := <-list.OnRelease(); releasedStreamName != "test1" {
				g.Errorf("released stream name is not correct, got %s, should be test1", releasedStreamName)
			}
		})

		g.It("Should return error when trying to close stream that does not exist", func() {
			list := newReadersList()

			list.Created("test1")
			list.Created("test2")
			err := list.Closed("unknown")

			g.Assert(err).Equal(errReaderDoesNotExist)
		})
	})
}
