package llama

import (
	"bytes"
	"testing"
)

func TestReadPieceGrowsOnNegativeReturn(t *testing.T) {
	long := bytes.Repeat([]byte("="), 300)
	calls := 0
	fill := func(p []byte) int {
		calls++
		if len(p) < len(long) {
			return -len(long)
		}
		return copy(p, long)
	}
	text, buf := readPiece(make([]byte, 256), fill)
	if !bytes.Equal(text, long) {
		t.Fatalf("got %d bytes, want %d", len(text), len(long))
	}
	if len(buf) != 300 || calls != 2 {
		t.Fatalf("buf=%d calls=%d, want 300 and 2", len(buf), calls)
	}
}

func TestReadPieceNoGrowForShortToken(t *testing.T) {
	fill := func(p []byte) int { return copy(p, "hi") }
	text, buf := readPiece(make([]byte, 256), fill)
	if string(text) != "hi" || len(buf) != 256 {
		t.Fatalf("text=%q buflen=%d", text, len(buf))
	}
}

func TestReadPieceEmpty(t *testing.T) {
	text, _ := readPiece(make([]byte, 256), func(p []byte) int { return 0 })
	if len(text) != 0 {
		t.Fatal("expected empty")
	}
}
