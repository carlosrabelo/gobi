package repl

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/carlosrabelo/gobi/gobi/pkg/term"
)

type pictureSlot struct {
	editable bool
	kind     rune
	literal  byte
}

type pictureSpec struct {
	slots []pictureSlot
	width int
}

func buildPictureSpec(picture string) pictureSpec {
	if picture == "" {
		return pictureSpec{}
	}

	spec := pictureSpec{width: len(picture)}
	for _, ch := range picture {
		switch ch {
		case '9', 'X', 'x', '!', '#', 'L', 'A', 'N':
			spec.slots = append(spec.slots, pictureSlot{editable: true, kind: ch})
		default:
			spec.slots = append(spec.slots, pictureSlot{editable: false, literal: byte(ch)})
		}
	}
	return spec
}

func (spec pictureSpec) editableCount() int {
	n := 0
	for _, slot := range spec.slots {
		if slot.editable {
			n++
		}
	}
	return n
}

func overlayPictureValue(spec pictureSpec, value string) string {
	if spec.width == 0 {
		return value
	}

	out := make([]byte, spec.width)
	valIdx := 0
	for i, slot := range spec.slots {
		if !slot.editable {
			out[i] = slot.literal
			continue
		}
		if valIdx < len(value) {
			out[i] = value[valIdx]
			valIdx++
			continue
		}
		out[i] = ' '
	}
	return string(out)
}

func validateReadValue(value, picture string) error {
	spec := buildPictureSpec(picture)
	if spec.width == 0 {
		return nil
	}

	display := overlayPictureValue(spec, value)
	valIdx := 0
	for i, slot := range spec.slots {
		if !slot.editable {
			continue
		}

		ch := display[i]
		if ch == ' ' {
			valIdx++
			continue
		}

		if err := validatePictureChar(slot.kind, ch); err != nil {
			return fmt.Errorf("invalid character %q at position %d: %w", string(ch), i+1, err)
		}
		valIdx++
	}

	if len(value) > spec.editableCount() {
		return fmt.Errorf("value too long for PICTURE")
	}

	return nil
}

func validatePictureChar(kind rune, ch byte) error {
	switch kind {
	case '9', '#':
		if ch < '0' || ch > '9' {
			return fmt.Errorf("expected digit")
		}
	case '!', 'A':
		if !unicode.IsLetter(rune(ch)) {
			return fmt.Errorf("expected letter")
		}
	case 'L':
		switch unicode.ToUpper(rune(ch)) {
		case 'T', 'F', 'Y', 'N':
		default:
			return fmt.Errorf("expected logical value")
		}
	case 'N':
		if (ch < '0' || ch > '9') && ch != '-' && ch != '+' && ch != ' ' {
			return fmt.Errorf("expected numeric character")
		}
	case 'X', 'x':
		if ch < 32 {
			return fmt.Errorf("expected printable character")
		}
	default:
		if ch < 32 {
			return fmt.Errorf("expected printable character")
		}
	}
	return nil
}

func pictureInsertChar(spec pictureSpec, value string, slotIdx int, ch byte) (string, int, error) {
	if spec.width == 0 {
		return insertPlainValue(value, ch, 255), len(value) + 1, nil
	}

	editable := editableSlotIndexes(spec)
	if len(editable) == 0 {
		return value, slotIdx, nil
	}
	if slotIdx < 0 {
		slotIdx = 0
	}
	if slotIdx >= len(editable) {
		slotIdx = len(editable) - 1
	}

	kind := spec.slots[editable[slotIdx]].kind
	normalized, err := normalizePictureInput(kind, ch)
	if err != nil {
		return value, slotIdx, err
	}

	runes := []byte(value)
	for len(runes) <= slotIdx {
		runes = append(runes, ' ')
	}
	if slotIdx < len(runes) {
		runes[slotIdx] = normalized
	} else {
		runes = append(runes, normalized)
	}

	next := slotIdx
	if next < len(editable)-1 {
		next++
	}
	return strings.TrimRight(string(runes), " "), next, nil
}

func pictureDeleteChar(spec pictureSpec, value string, slotIdx int) (string, int) {
	if spec.width == 0 {
		if len(value) == 0 {
			return value, 0
		}
		if slotIdx > len(value) {
			slotIdx = len(value)
		}
		if slotIdx == 0 {
			return value, 0
		}
		return value[:slotIdx-1] + value[slotIdx:], slotIdx - 1
	}

	editable := editableSlotIndexes(spec)
	if len(editable) == 0 || slotIdx <= 0 {
		return value, 0
	}

	runes := []byte(strings.TrimRight(value, " "))
	for len(runes) <= slotIdx {
		runes = append(runes, ' ')
	}
	for i := slotIdx; i < len(runes); i++ {
		if i+1 < len(runes) {
			runes[i] = runes[i+1]
		} else {
			runes[i] = ' '
		}
	}
	return strings.TrimRight(string(runes), " "), slotIdx - 1
}

func editableSlotIndexes(spec pictureSpec) []int {
	indexes := make([]int, 0, spec.editableCount())
	for i, slot := range spec.slots {
		if slot.editable {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func cursorColumnForPicture(spec pictureSpec, col, slotIdx int) int {
	if spec.width == 0 {
		return col + slotIdx
	}

	editable := editableSlotIndexes(spec)
	if slotIdx < 0 {
		slotIdx = 0
	}
	if slotIdx >= len(editable) {
		slotIdx = len(editable) - 1
	}
	if len(editable) == 0 {
		return col
	}
	return col + editable[slotIdx]
}

func normalizePictureInput(kind rune, ch byte) (byte, error) {
	if ch < 32 {
		return 0, fmt.Errorf("non-printable character")
	}

	switch kind {
	case '9', '#':
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("expected digit")
		}
		return ch, nil
	case '!', 'A':
		if !unicode.IsLetter(rune(ch)) {
			return 0, fmt.Errorf("expected letter")
		}
		return byte(unicode.ToUpper(rune(ch))), nil
	case 'L':
		switch unicode.ToUpper(rune(ch)) {
		case 'T', 'Y':
			return 'T', nil
		case 'F', 'N':
			return 'F', nil
		default:
			return 0, fmt.Errorf("expected logical value")
		}
	case 'N':
		if (ch >= '0' && ch <= '9') || ch == '-' || ch == '+' {
			return ch, nil
		}
		return 0, fmt.Errorf("expected numeric character")
	default:
		return ch, nil
	}
}

func insertPlainValue(value string, ch byte, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 255
	}
	if len(value) >= maxLen {
		return value
	}
	return value + string(ch)
}

func readFieldWidth(screen *term.Screen, field term.GetField, spec pictureSpec) int {
	if spec.width > 0 {
		return spec.width
	}
	remaining := screen.Cols() - field.Col
	if remaining <= 0 {
		return 1
	}
	return remaining
}
