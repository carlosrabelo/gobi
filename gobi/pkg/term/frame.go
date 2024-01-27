package term

import (
	"bytes"
	"io"
)

// FrameWriter composes terminal frames in a back buffer and presents them
// to the output device without flicker from incremental drawing.
type FrameWriter struct {
	out   io.Writer
	back  bytes.Buffer
	front []byte
}

// NewFrameWriter returns a double-buffered frame writer for out.
func NewFrameWriter(out io.Writer) *FrameWriter {
	return &FrameWriter{out: out}
}

// Begin resets the back buffer for a new frame.
func (f *FrameWriter) Begin() {
	if f == nil {
		return
	}
	f.back.Reset()
}

// Back returns the back buffer used to compose the next frame.
func (f *FrameWriter) Back() io.Writer {
	if f == nil {
		return io.Discard
	}
	return &f.back
}

// Present writes the back buffer to the terminal when it differs from the
// previously presented frame.
func (f *FrameWriter) Present() error {
	if f == nil {
		return nil
	}

	frame := append([]byte(nil), f.back.Bytes()...)
	if f.front != nil && bytes.Equal(frame, f.front) {
		return nil
	}

	if err := ClearScreen(f.out); err != nil {
		return err
	}
	if len(frame) > 0 {
		if _, err := f.out.Write(frame); err != nil {
			return err
		}
	}

	f.front = make([]byte, len(frame))
	copy(f.front, frame)
	return nil
}

// Reset clears both buffers and forgets the last presented frame.
func (f *FrameWriter) Reset() {
	if f == nil {
		return
	}
	f.back.Reset()
	f.front = nil
}
