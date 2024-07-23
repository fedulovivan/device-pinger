package utils

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TruncateSuite struct {
	suite.Suite
}

func (s *TruncateSuite) Test1() {
	input := "lorem ipsum"
	actual := Truncate(input, 5)
	expected := "lorem"
	s.Equal(expected, actual)
}

func (s *TruncateSuite) Test2() {
	input := "lor"
	actual := Truncate(input, 5)
	expected := "lor"
	s.Equal(expected, actual)
}

func (s *TruncateSuite) Test3() {
	input := "lor    "
	actual := Truncate(input, 4)
	expected := "lor "
	s.Equal(expected, actual)
}

func (s *TruncateSuite) Test4() {
	input := "AB"
	actual := Truncate(input, 1)
	expected := "A"
	s.Equal(expected, actual)
}

func (s *TruncateSuite) Test5() {
	input := "ЙЫ"
	actual := Truncate(input, 1)
	expected := "Й"
	s.Equal(expected, actual)
}

func (s *TruncateSuite) Test6() {
	input := "你好"
	actual := Truncate(input, 1)
	expected := "你"
	s.Equal(expected, actual)
}

func (s *TruncateSuite) Test7() {
	input := "蒙古兄弟你好"
	actual := Truncate(input, 3)
	expected := "蒙古兄"
	s.Equal(expected, actual)
}

func TestTruncate(t *testing.T) {
	suite.Run(t, new(TruncateSuite))
}
