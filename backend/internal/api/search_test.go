package api

import (
	"reflect"
	"testing"
)

func TestSearchTermsSanitizeFTSInput(t *testing.T) {
	got := searchTerms(`docker:"ps" OR password=secret; apt-get`, 8)
	want := []string{"docker", "ps", "or", "password", "secret", "apt", "get"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected terms: %#v", got)
	}
}

func TestBuildFTSQueryLimitsTerms(t *testing.T) {
	got := buildFTSQuery("one two three four five six seven eight nine ten")
	want := "one two three four five six seven eight"
	if got != want {
		t.Fatalf("unexpected query %q, want %q", got, want)
	}
}

func TestBuildFTSQueryReturnsEmptyForPunctuationOnly(t *testing.T) {
	if got := buildFTSQuery(`"":;()--`); got != "" {
		t.Fatalf("expected empty query, got %q", got)
	}
}
