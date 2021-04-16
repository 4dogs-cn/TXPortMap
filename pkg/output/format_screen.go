package output

import (
	"bytes"
	"github.com/4dogs-cn/TXPortMap/pkg/Ginfo/Ghttp"
	"github.com/4dogs-cn/TXPortMap/pkg/conversion"
	"github.com/fatih/color"
)

// formatScreen formats the output for showing on screen.
func (w *StandardWriter) formatScreen(output *ResultEvent) []byte {
	builder := &bytes.Buffer{}
	builder.WriteRune('[')
	builder.WriteString(color.CyanString(output.Time.Format("2006-01-02 15:04:05")))
	builder.WriteString("] ")
	builder.WriteRune('[')
	builder.WriteString(color.RedString(output.Target))
	builder.WriteString("] ")
	builder.WriteRune('[')
	builder.WriteString(color.YellowString(output.Info.Service))
	builder.WriteString("] ")

	if output.WorkingEvent != nil {
		switch tmp := output.WorkingEvent.(type) {
		case Ghttp.Result:
			builder.WriteString(tmp.ToString())
		default:
			builder.WriteString(conversion.ToString(tmp))
		}
	} else {
		builder.WriteRune('[')
		builder.WriteString(color.GreenString(output.Info.Banner))
		builder.WriteString("] ")
	}
	return builder.Bytes()
}
