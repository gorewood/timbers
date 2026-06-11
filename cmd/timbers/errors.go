// Package main provides the entry point for the timbers CLI.
package main

import (
	"fmt"
	"io"

	"github.com/charmbracelet/fang"
)

// newErrorHandler returns a fang error handler that stays legible under piping
// and output compression.
//
// fang's DefaultErrorHandler is supposed to print a plain, unstyled line when
// stderr is not a terminal — but fang.Execute wraps stderr in a
// colorprofile.Writer before invoking the handler, so DefaultErrorHandler's
// `w.(term.File)` check fails on the wrapper and the non-TTY path is never
// taken. The result is that the multi-line, padded "ERROR" box is rendered even
// into a pipe, and because that box ends in a blank line, `cmd ... 2>&1 |
// tail -1` and output compressors crop the failure down to an empty line — a
// blocked commit (e.g. the pre-commit "undocumented commits" gate) then looks
// indistinguishable from success.
//
// We do the TTY decision ourselves against the real stderr: a terminal keeps
// fang's styled box; anything else (pipe, file, compressor) gets the error on a
// single plain line so it survives truncation.
func newErrorHandler(stderrIsTTY bool) fang.ErrorHandler {
	return func(w io.Writer, styles fang.Styles, err error) {
		if stderrIsTTY {
			fang.DefaultErrorHandler(w, styles, err)
			return
		}
		_, _ = fmt.Fprintln(w, err.Error())
	}
}
