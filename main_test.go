package main

import "testing"

func TestReadlineInputFilterTurnsPastedNewlineIntoSpace(t *testing.T) {
	filter := newReadlineInputFilter()

	got, ok := filter('\n')
	if !ok {
		t.Fatal("newline should still be processed")
	}
	if got != ' ' {
		t.Fatalf("newline = %q, want space", got)
	}
}

func TestReadlineInputFilterKeepsEnterAsSubmit(t *testing.T) {
	filter := newReadlineInputFilter()

	got, ok := filter('\r')
	if !ok {
		t.Fatal("carriage return should still be processed")
	}
	if got != '\r' {
		t.Fatalf("carriage return = %q, want unchanged", got)
	}
}

func TestReadlineInputFilterKeepsNormalRunes(t *testing.T) {
	filter := newReadlineInputFilter()

	got, ok := filter('尤')
	if !ok {
		t.Fatal("normal rune should still be processed")
	}
	if got != '尤' {
		t.Fatalf("normal rune = %q, want unchanged", got)
	}
}
