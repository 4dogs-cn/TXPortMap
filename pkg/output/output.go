package output

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/logrusorgru/aurora"
	"github.com/pkg/errors"
	"os"
	"regexp"
	"sync"
	"time"
)

// Writer is an interface which writes output to somewhere for nuclei events.
type Writer interface {
	// Close closes the output writer interface
	Close()
	// Colorizer returns the colorizer instance for writer
	Colorizer() aurora.Aurora
	// Write writes the event to file and/or screen.
	Write(*ResultEvent) error
	// Request logs a request in the trace log
	Request(ip, port, requestType string, err error)
}

type Info struct {
	Banner  string
	Service string
}
type ResultEvent struct {
	WorkingEvent string    `json:"WorkingEvent"`
	Info         Info      `json:"info,inline"`
	Time         time.Time `json:"time"`
	Target       string    `json:"Target"`
}

type StandardWriter struct {
	json        bool
	aurora      aurora.Aurora
	outputFile  *fileWriter
	outputMutex *sync.Mutex
	traceFile   *fileWriter
	traceMutex  *sync.Mutex
	Colors      *Colorizer
}

var decolorizerRegex = regexp.MustCompile(`\x1B\[[0-9;]*[a-zA-Z]`)

func NewStandardWriter(color, json bool, file, traceFile string) (*StandardWriter, error) {
	auroraColorizer := aurora.NewAurora(color)

	var outputFile *fileWriter
	if file != "" {
		output, err := newFileOutputWriter(file)
		if err != nil {
			return nil, errors.Wrap(err, "could not create output file")

		}
		outputFile = output
	}
	var traceOutput *fileWriter
	if traceFile != "" {
		output, err := newFileOutputWriter(traceFile)
		if err != nil {
			return nil, errors.Wrap(err, "could not create output file")
		}
		traceOutput = output
	}
	writer := &StandardWriter{
		json:        json,
		aurora:      auroraColorizer,
		outputFile:  outputFile,
		outputMutex: &sync.Mutex{},
		traceFile:   traceOutput,
		traceMutex:  &sync.Mutex{},
		Colors:      ColorNew(auroraColorizer),
	}
	return writer, nil
}

// Write writes the event to file and/or screen.
func (w *StandardWriter) Write(event *ResultEvent) error {
	event.Time = time.Now()

	var data []byte
	var err error
	if w.json {
		data, err = w.formatJSON(event)
	} else {
		data = w.formatScreen(event)
	}

	if err != nil {
		return errors.Wrap(err, "could not format output")
	}
	if len(data) == 0 {
		return nil
	}
	_, _ = os.Stdout.Write(data)
	_, _ = os.Stdout.Write([]byte("\n"))
	if w.outputFile != nil {
		if !w.json {
			data = decolorizerRegex.ReplaceAll(data, []byte(""))
		}
		if writeErr := w.outputFile.Write(data); writeErr != nil {
			return errors.Wrap(err, "could not write to output")
		}
	}
	return nil
}

func (w *StandardWriter) Colorizer() aurora.Aurora {
	return w.aurora
}

func (w *StandardWriter) Close() {
	if w.outputFile != nil {
		w.outputFile.Close()
	}
	if w.traceFile != nil {
		w.traceFile.Close()
	}
}

type JSONTraceRequest struct {
	IP    string `json:"ip"`
	Port  string `json:"port"`
	Error string `json:"error"`
	Type  string `json:"type"`
}

// Request writes a log the requests trace log
func (w *StandardWriter) Request(ip, port, requestType string, err error) {
	if w.traceFile == nil {
		return
	}
	request := &JSONTraceRequest{
		IP:   ip,
		Port: port,
		Type: requestType,
	}
	if err != nil {
		request.Error = err.Error()
	} else {
		request.Error = "none"
	}

	data, err := jsoniter.Marshal(request)
	if err != nil {
		return
	}
	w.traceMutex.Lock()
	_ = w.traceFile.Write(data)
	w.traceMutex.Unlock()
}
