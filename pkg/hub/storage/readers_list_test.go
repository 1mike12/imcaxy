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
			list.Closed("test")

			if len(list.OnRelease()) != 1 {
				g.Errorf("released stream name not sent on stream release")
			}

			if releasedStreamName := <-list.OnRelease(); releasedStreamName != "test" {
				g.Errorf("released stream name is not correct")
			}
		})

		g.It("Should not return error on close if stream exists", func() {
			list := newReadersList()

			list.Created("test")
			err := list.Closed("test")

			g.Assert(err).IsNil()
		})

		g.It("Should notify about stream release when multiple streams registered", func() {
			list := newReadersList()

			list.Created("test1")
			list.Created("test2")
			list.Closed("test1")

			if len(list.OnRelease()) < 1 {
				g.Errorf("released stream name not sent on stream release, len of chan is %v, should be 1", len(list.OnRelease()))
			}

			if len(list.OnRelease()) > 1 {
				g.Errorf("too many streams released, len of chan is %v, should be 1", len(list.OnRelease()))
			}

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
