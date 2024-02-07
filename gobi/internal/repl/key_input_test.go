package repl

import (
	"testing"

	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

func TestReplKeyByte(t *testing.T) {
	tests := []struct {
		name string
		key  term.Key
		want byte
	}{
		{name: "byte", key: term.Key{Kind: term.KeyByte, Byte: 'X'}, want: 'X'},
		{name: "enter", key: term.Key{Kind: term.KeyEnter}, want: editKeyEnter},
		{name: "backspace", key: term.Key{Kind: term.KeyBackspace}, want: editKeyDel},
		{name: "shift tab", key: term.Key{Kind: term.KeyShiftTab}, want: editKeyCtrlK},
		{name: "escape", key: term.Key{Kind: term.KeyEscape}, want: editKeyEsc},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := replKeyByte(tc.key)
			if err != nil {
				t.Fatalf("replKeyByte: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}
