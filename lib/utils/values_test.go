package utils

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ValuesSuite struct {
	suite.Suite
}

func (s *ValuesSuite) Test1() {
	input := map[string]int{"Foo": 1, "Bar": 2}
	actual := Values(input)
	expected := []int{1, 2}
	s.ElementsMatch(actual, expected)
}

func (s *ValuesSuite) Test2() {
	type Ipsum struct {
		foo int
		bar string
	}
	input := map[string]Ipsum{
		"Lorem": {foo: 1, bar: "baz"},
	}
	actual := Values(input)
	expected := []Ipsum{{foo: 1, bar: "baz"}}
	s.ElementsMatch(actual, expected)
}

func TestValues(t *testing.T) {
	suite.Run(t, new(ValuesSuite))
}
