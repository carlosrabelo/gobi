package hostterm

import "testing"

func TestDecodeANSIKeyArrowsAndEnter(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"\r", KeyEnter},
		{"\n", KeyEnter},
		{"\x1b[A", KeyUp},
		{"\x1b[B", KeyDown},
		{"\x1b[C", KeyRight},
		{"\x1b[D", KeyLeft},
		{"\x1bOA", KeyUp},
		{"\x1b[5~", KeyPgUp},
		{"\x1b[6~", KeyPgDn},
		{"a", 'a'},
	}
	for _, tc := range cases {
		got, err := DecodeANSIKeyBytes([]byte(tc.in))
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("%q: got %d want %d", tc.in, got, tc.want)
		}
	}
}
