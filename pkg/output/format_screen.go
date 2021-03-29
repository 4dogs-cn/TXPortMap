package output

import (
	"bytes"
	"github.com/4dogs-cn/TXPortMap/pkg/Ginfo/Ghttp"
	"github.com/4dogs-cn/TXPortMap/pkg/conversion"
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

	if output.WorkingEvent != nil{
		switch tmp := output.WorkingEvent.(type) {
		case Ghttp.Result:
			builder.WriteString(tmp.ToString(w.aurora))
		default:
			builder.WriteString(conversion.ToString(tmp))
		}
	}else{
		builder.WriteRune('[')
		builder.WriteString(w.aurora.Yellow(output.Info.Service).String())
		builder.WriteString(": ")
		builder.WriteString(w.aurora.Green(output.Info.Banner).String())
		builder.WriteString("] ")
	}
	return builder.Bytes()
}
