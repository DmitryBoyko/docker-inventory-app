package app

import "testing"

func TestLooksLikeStdcopy(t *testing.T) {
	if !looksLikeStdcopy([]byte{1, 0, 0, 0, 0, 0, 0, 5}) {
		t.Fatal("expected stdcopy header")
	}
	if looksLikeStdcopy([]byte("hello wo")) {
		t.Fatal("plain text should not look like stdcopy")
	}
	if looksLikeStdcopy([]byte{1, 0}) {
		t.Fatal("short peek")
	}
}
