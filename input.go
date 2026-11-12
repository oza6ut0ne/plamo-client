package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// errInterrupt is returned by ReadBlock when the user presses Ctrl+C.
var errInterrupt = errors.New("interrupt")

// writeRawLine writes s followed by a newline to w. While interactive mode runs,
// the terminal is in raw mode and output post-processing is disabled, so
// embedded newlines are converted to CRLF to keep every line starting at
// column 0.
func writeRawLine(w io.Writer, s string) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "\r\n")
	fmt.Fprint(w, s, "\r\n")
}

// keyKind classifies a decoded terminal input event.
type keyKind int

const (
	keyRune keyKind = iota
	keyUp
	keyDown
	keyLeft
	keyRight
	keyHome
	keyEnd
	keyDelete
	keyPasteStart
	keyPasteEnd
	keyUnknown
)

// keyEvent is a single decoded key press.
type keyEvent struct {
	kind keyKind
	r    rune
}

// keyReader yields decoded key events. More reports whether another event is
// immediately available, which is used to detect a pasted block on terminals
// that do not support bracketed paste.
type keyReader interface {
	Next() (keyEvent, error)
	More() bool
}

// terminalKeyReader decodes key events from a terminal input stream.
type terminalKeyReader struct {
	br *bufio.Reader
}

func newTerminalKeyReader(r io.Reader) *terminalKeyReader {
	return &terminalKeyReader{br: bufio.NewReader(r)}
}

func (r *terminalKeyReader) Next() (keyEvent, error) {
	c, _, err := r.br.ReadRune()
	if err != nil {
		return keyEvent{}, err
	}
	if c != 0x1b {
		return keyEvent{kind: keyRune, r: c}, nil
	}
	// A lone ESC has no following bytes buffered. An escape sequence (arrow
	// key, bracketed paste marker, ...) is written atomically, so the whole
	// sequence is already buffered.
	if r.br.Buffered() == 0 {
		return keyEvent{kind: keyUnknown}, nil
	}
	prefix, err := r.br.ReadByte()
	if err != nil {
		return keyEvent{}, err
	}
	if prefix != '[' && prefix != 'O' {
		return keyEvent{kind: keyUnknown}, nil
	}
	var seq []byte
	for {
		b, err := r.br.ReadByte()
		if err != nil {
			return keyEvent{}, err
		}
		seq = append(seq, b)
		if b >= 0x40 && b <= 0x7e {
			break
		}
		if len(seq) >= 16 {
			break
		}
	}
	return decodeCSI(prefix, seq), nil
}

func (r *terminalKeyReader) More() bool {
	return r.br.Buffered() > 0
}

func decodeCSI(prefix byte, seq []byte) keyEvent {
	if prefix == 'O' {
		switch string(seq) {
		case "A":
			return keyEvent{kind: keyUp}
		case "B":
			return keyEvent{kind: keyDown}
		case "C":
			return keyEvent{kind: keyRight}
		case "D":
			return keyEvent{kind: keyLeft}
		case "H":
			return keyEvent{kind: keyHome}
		case "F":
			return keyEvent{kind: keyEnd}
		}
		return keyEvent{kind: keyUnknown}
	}
	switch string(seq) {
	case "A":
		return keyEvent{kind: keyUp}
	case "B":
		return keyEvent{kind: keyDown}
	case "C":
		return keyEvent{kind: keyRight}
	case "D":
		return keyEvent{kind: keyLeft}
	case "H", "1~":
		return keyEvent{kind: keyHome}
	case "F", "4~":
		return keyEvent{kind: keyEnd}
	case "3~":
		return keyEvent{kind: keyDelete}
	case "200~":
		return keyEvent{kind: keyPasteStart}
	case "201~":
		return keyEvent{kind: keyPasteEnd}
	}
	return keyEvent{kind: keyUnknown}
}

// lineEditor implements a small append/insert line editor on top of a sequence
// of key events. It exists so that a pasted block (which contains newlines) can
// be collected into a single input instead of being submitted line by line.
type lineEditor struct {
	prompt   []rune
	line     []rune
	pos      int
	width    int
	out      io.Writer
	lastRows int
	lastCur  int
	history  []string
	histIdx  int
}

func newLineEditor(prompt string, width int, out io.Writer) *lineEditor {
	if width <= 0 {
		width = 80
	}
	return &lineEditor{
		prompt: []rune(prompt),
		width:  width,
		out:    out,
	}
}

func (e *lineEditor) insert(r rune) {
	e.line = append(e.line, 0)
	copy(e.line[e.pos+1:], e.line[e.pos:])
	e.line[e.pos] = r
	e.pos++
}

func (e *lineEditor) backspace() {
	if e.pos == 0 {
		return
	}
	copy(e.line[e.pos-1:], e.line[e.pos:])
	e.line = e.line[:len(e.line)-1]
	e.pos--
}

func (e *lineEditor) deleteForward() {
	if e.pos >= len(e.line) {
		return
	}
	copy(e.line[e.pos:], e.line[e.pos+1:])
	e.line = e.line[:len(e.line)-1]
}

func (e *lineEditor) setLine(s string) {
	e.line = []rune(s)
	e.pos = len(e.line)
}

func (e *lineEditor) reset() {
	e.line = e.line[:0]
	e.pos = 0
	e.histIdx = len(e.history)
	e.lastRows, e.lastCur = 0, 0
}

// displayString converts the edited text into the bytes to write to the
// terminal. Newlines are emitted as CRLF because raw mode disables output
// post-processing.
func (e *lineEditor) displayString(text []rune) string {
	var b strings.Builder
	for _, r := range text {
		if r == '\n' {
			b.WriteString("\r\n")
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (e *lineEditor) render() {
	if e.out == nil {
		return
	}
	if e.lastCur > 0 {
		fmt.Fprintf(e.out, "\x1b[%dA", e.lastCur)
	}
	fmt.Fprint(e.out, "\r\x1b[J")
	text := make([]rune, 0, len(e.prompt)+len(e.line))
	text = append(text, e.prompt...)
	text = append(text, e.line...)
	fmt.Fprint(e.out, e.displayString(text))
	endRow, _ := terminalPos(text, e.width)
	curRow, curCol := terminalPos(text[:len(e.prompt)+e.pos], e.width)
	if endRow > curRow {
		fmt.Fprintf(e.out, "\x1b[%dA", endRow-curRow)
	}
	fmt.Fprint(e.out, "\r")
	if curCol > 0 {
		fmt.Fprintf(e.out, "\x1b[%dC", curCol)
	}
	e.lastRows = endRow + 1
	e.lastCur = curRow
}

func (e *lineEditor) submit() string {
	if len(e.line) > 0 {
		e.history = append(e.history, string(e.line))
	}
	if e.out != nil {
		fmt.Fprint(e.out, "\r\n")
	}
	e.lastRows, e.lastCur = 0, 0
	return string(e.line)
}

// ReadBlock reads one input from src and returns it. A bracketed paste is
// collected as a single input with its newlines preserved. When the terminal
// does not delimit pastes, a newline followed by more buffered input is also
// treated as part of a pasted block.
func (e *lineEditor) ReadBlock(src keyReader) (string, error) {
	e.reset()
	e.render()

	pasting := false
	lastWasCR := false

	for {
		ev, err := src.Next()
		if err != nil {
			return "", err
		}

		if pasting {
			switch ev.kind {
			case keyPasteEnd:
				pasting = false
				lastWasCR = false
				e.render()
			case keyRune:
				switch ev.r {
				case '\r':
					e.insert('\n')
					lastWasCR = true
				case '\n':
					if !lastWasCR {
						e.insert('\n')
					}
					lastWasCR = false
				default:
					e.insert(ev.r)
					lastWasCR = false
				}
			}
			continue
		}

		switch ev.kind {
		case keyPasteStart:
			pasting = true
			lastWasCR = false
		case keyPasteEnd:
			// Ignore a stray paste-end marker.
		case keyRune:
			switch ev.r {
			case '\r', '\n':
				if src.More() {
					e.insert('\n')
					e.render()
					continue
				}
				return e.submit(), nil
			case 0x03: // Ctrl+C
				return "", errInterrupt
			case 0x04: // Ctrl+D
				if len(e.line) == 0 {
					return "", io.EOF
				}
				e.deleteForward()
				e.render()
			case 0x7f, 0x08: // Backspace
				e.backspace()
				e.render()
			case 0x15: // Ctrl+U
				e.line = e.line[:0]
				e.pos = 0
				e.render()
			case 0x01: // Ctrl+A
				e.pos = 0
				e.render()
			case 0x05: // Ctrl+E
				e.pos = len(e.line)
				e.render()
			case 0x02: // Ctrl+B
				if e.pos > 0 {
					e.pos--
					e.render()
				}
			case 0x06: // Ctrl+F
				if e.pos < len(e.line) {
					e.pos++
					e.render()
				}
			case 0x0c: // Ctrl+L
				fmt.Fprint(e.out, "\x1b[2J\x1b[H")
				e.lastRows, e.lastCur = 0, 0
				e.render()
			case '\t':
				// Ignore tabs; they do not have a well-defined display width.
			default:
				if ev.r >= 0x20 && ev.r != 0x7f {
					e.insert(ev.r)
					e.render()
				}
			}
		case keyUp:
			if e.histIdx > 0 {
				e.histIdx--
				e.setLine(e.history[e.histIdx])
				e.render()
			}
		case keyDown:
			if e.histIdx < len(e.history) {
				e.histIdx++
				if e.histIdx == len(e.history) {
					e.setLine("")
				} else {
					e.setLine(e.history[e.histIdx])
				}
				e.render()
			}
		case keyLeft:
			if e.pos > 0 {
				e.pos--
				e.render()
			}
		case keyRight:
			if e.pos < len(e.line) {
				e.pos++
				e.render()
			}
		case keyHome:
			e.pos = 0
			e.render()
		case keyEnd:
			e.pos = len(e.line)
			e.render()
		case keyDelete:
			e.deleteForward()
			e.render()
		}
	}
}

// terminalPos returns the 0-based row and column at which the terminal cursor
// rests after writing text, taking soft wrapping and embedded newlines into
// account.
func terminalPos(text []rune, width int) (row, col int) {
	if width <= 0 {
		width = 80
	}
	for _, r := range text {
		switch r {
		case '\n':
			row++
			col = 0
			continue
		case '\t':
			col = (col/8 + 1) * 8
			if col >= width {
				row++
				col = 0
			}
			continue
		}
		w := runewidth.RuneWidth(r)
		if w <= 0 {
			continue
		}
		if col+w > width {
			row++
			col = 0
		}
		col += w
		if col >= width {
			row++
			col = 0
		}
	}
	return row, col
}

// interactiveReader owns the terminal while interactive translation runs.
type interactiveReader struct {
	in     *os.File
	out    *os.File
	old    *term.State
	src    keyReader
	editor *lineEditor
}

func newInteractiveReader(in, out *os.File) (*interactiveReader, error) {
	old, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return nil, err
	}
	width, _, err := term.GetSize(int(out.Fd()))
	if err != nil || width <= 0 {
		width = 80
	}
	ir := &interactiveReader{
		in:  in,
		out: out,
		old: old,
		src: newTerminalKeyReader(in),
	}
	ir.editor = newLineEditor("plamo> ", width, out)
	// Enable bracketed paste so that pasted newlines are distinguishable from
	// the Enter key.
	fmt.Fprint(out, "\x1b[?2004h")
	return ir, nil
}

func (ir *interactiveReader) ReadBlock() (string, error) {
	return ir.editor.ReadBlock(ir.src)
}

func (ir *interactiveReader) Close() error {
	fmt.Fprint(ir.out, "\x1b[?2004l")
	if ir.old != nil {
		err := term.Restore(int(ir.in.Fd()), ir.old)
		ir.old = nil
		return err
	}
	return nil
}
