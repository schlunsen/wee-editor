package wtfwyt

import (
	"io"
	"strings"
)

// io_TeeReader mirrors io.TeeReader with a strings.Builder sink, so tests can
// assert on the exact bytes transmitted rather than on a re-encoded struct.
func io_TeeReader(r io.Reader, w *strings.Builder) io.Reader {
	return io.TeeReader(r, w)
}
