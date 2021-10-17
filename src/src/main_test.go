package main

import (
	"testing"

	. "github.com/franela/goblin"
)

func TestCalculator(t *testing.T) {
	g := Goblin(t)
	g.Describe("Calculator", func() {
		g.It("should add two numbers", func() {
			g.Assert(1 + 1).Equal(2)
		})

		g.It("should substract two numbers", func() {
			g.Assert(1 - 1).Equal(1)
		})

		g.It("should multiply two numbers")
	})
}
