package config

import (
	"flag"
	"reflect"
	"testing"
)

func TestStringSet_Set(t *testing.T) {
	t.Parallel()
	var s StringSet
	for _, v := range []string{"a", "b", "a", "c", ""} {
		if err := s.Set(v); err != nil {
			t.Fatalf("Set(%q): %v", v, err)
		}
	}
	if !reflect.DeepEqual([]string(s), []string{"a", "b", "c"}) {
		t.Errorf("StringSet = %v, want [a b c]", s)
	}
}

func TestStringSet_Contains(t *testing.T) {
	t.Parallel()
	s := StringSet{"a", "b"}
	if !s.Contains("a") {
		t.Error("Contains(a) should be true")
	}
	if s.Contains("c") {
		t.Error("Contains(c) should be false")
	}
}

func TestStringSet_String(t *testing.T) {
	t.Parallel()
	s := StringSet{"a", "b"}
	if got := s.String(); got != "a,b" {
		t.Errorf("String() = %q, want a,b", got)
	}
	var nilSet *StringSet
	if got := nilSet.String(); got != "" {
		t.Errorf("nil String() = %q, want empty", got)
	}
}

func TestStringSet_ImplementsFlagValue(t *testing.T) {
	t.Parallel()
	var s StringSet
	var _ flag.Value = &s
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.Var(&s, "x", "")
	if err := fs.Parse([]string{"--x=a", "--x=b", "--x=a"}); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !reflect.DeepEqual([]string(s), []string{"a", "b"}) {
		t.Errorf("got %v, want [a b]", s)
	}
}
