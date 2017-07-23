package main

import (
	"testing"
)

var fts5_tests = []struct {
	src, expected string
}{
	{"foo", "\"foo\""},
	{"foo bar", "\"foo\" \"bar\""},
	{"\"foo\"", "\"foo\""},
	{"\"foo bar\"", "\"foo bar\""},
	{"foo AND bar", "\"foo\" AND \"bar\""},
	{"(foo AND bar) OR baz", "(\"foo\" AND \"bar\") OR \"baz\""},
	{"foo AN bar", "\"foo\" \"AN\" \"bar\""},
	{"\"foo AN bar\"", "\"foo AN bar\""},
}

func TestFTS5(t *testing.T) {
	for i, test := range fts5_tests {
		actual, err := fts5_term(test.src)
		if err != nil {
			t.Errorf("%d: %s expected %s got Error(%s)", i, test.src, test.expected, err)
		} else if actual != test.expected {
			t.Errorf("%d: %s expected %s got %s", i, test.src, test.expected, actual)
		}
	}
}
