package output

import (
	"testing"
	"time"
)

func TestNewStandardWriter(t *testing.T) {
	writer, err := NewStandardWriter(false, false, "scan.txt", "trace.log")
	if err !=nil{
		t.Logf("new writer error :%s\n",err.Error())
	}
	even := &ResultEvent{
		Target: "192.168.0.53",
		Time: time.Now(),
		WorkingEvent: "time out",
	}
	writer.Write(even)
}
