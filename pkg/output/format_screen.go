package output

import (
	"bytes"
)

// formatScreen formats the output for showing on screen.
func (w *StandardWriter) formatScreen(output *ResultEvent) []byte {
	builder := &bytes.Buffer{}
	builder.WriteRune('[')
	builder.WriteString(w.aurora.Cyan(output.Time.Format("2006-01-02 15:04:05")).String())
	builder.WriteString("] ")
	builder.WriteRune('[')
	builder.WriteString(w.aurora.Red(output.Target).String())
	builder.WriteString("] ")

	builder.WriteRune('[')
	builder.WriteString(w.aurora.Yellow(output.Info.Service).String())
	builder.WriteString(": ")
	builder.WriteString(w.aurora.Red(output.Info.Banner).String())
	builder.WriteString("] ")

	return builder.Bytes()
}
