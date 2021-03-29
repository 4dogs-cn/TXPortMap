package Ghttp

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/logrusorgru/aurora"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// HTTP defines the plain http scheme
	HTTP = "http"
	// HTTPS defines the secure http scheme
	HTTPS = "https"
	// HTTPorHTTPS defines the both http and https scheme
	HTTPorHTTPS = "http|https"
)

type ScanOptions struct {
	Methods                []string
	StoreResponseDirectory string
	RequestURI             string
	RequestBody            string
	VHost                  bool
	OutputTitle            bool
	OutputStatusCode       bool
	OutputLocation         bool
	OutputContentLength    bool
	StoreResponse          bool
	OutputServerHeader     bool
	OutputWebSocket        bool
	OutputWithNoColor      bool
	OutputMethod           bool
	ResponseInStdout       bool
	TLSProbe               bool
	CSPProbe               bool
	OutputContentType      bool
	Unsafe                 bool
	Pipeline               bool
	HTTP2Probe             bool
	OutputIP               bool
	OutputCName            bool
	OutputCDN              bool
	OutputResponseTime     bool
	PreferHTTPS            bool
	NoFallback             bool
}

func Analyze(protocol, domain string, port int, method string, scanopts *ScanOptions) Result {
	origProtocol := protocol
	if protocol == "http" {
		protocol = HTTP
	} else {
		protocol = HTTPS
	}
	retried := false
retry:
	URL := fmt.Sprintf("%s://%s", protocol, domain)
	if port > 0 {
		URL = fmt.Sprintf("%s://%s:%d", protocol, domain, port)
	}

	var client *http.Client
	//DEBUG := false
	//if DEBUG {
	//	proxyUrl := "http://127.0.0.1:8080"
	//	proxy, _ := url.Parse(proxyUrl)
	//	tr := &http.Transport{
	//		Proxy:           http.ProxyURL(proxy),
	//		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	//	}
	//	client = &http.Client{
	//		Transport: tr,               //proxy
	//		Timeout:   time.Second * 10, //timeout
	//	}
	//} else {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{
		Timeout:   time.Second * 10, //timeout
		Transport: tr,
	}
	//}

	req, err := http.NewRequest(method, URL, nil)
	if err != nil {
		return Result{URL: URL, err: err}
	}
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36")

	resp, err := client.Do(req)

	if err != nil {
		if !retried && origProtocol == HTTPorHTTPS {
			if protocol == HTTPS {
				protocol = HTTP
			} else {
				protocol = HTTPS
			}
			retried = true
			goto retry
		}
		return Result{URL: URL, err: err}
	}

	var fullURL string

	if resp.StatusCode >= 0 {
		if port > 0 {
			fullURL = fmt.Sprintf("%s://%s:%d", protocol, domain, port)
		} else {
			fullURL = fmt.Sprintf("%s://%s", protocol, domain)
		}
	}

	builder := &strings.Builder{}
	builder.WriteString(fullURL)

	if scanopts.OutputStatusCode {
		builder.WriteString(" [")
		builder.WriteString(strconv.Itoa(resp.StatusCode))
		builder.WriteRune(']')
	}

	if scanopts.OutputContentLength {
		builder.WriteString(" [")
		builder.WriteString(strconv.FormatInt(resp.ContentLength, 10))
		builder.WriteRune(']')
	}

	if scanopts.OutputContentType {
		builder.WriteString(" [")
		builder.WriteString(resp.Header.Get("Content-Type"))
		builder.WriteRune(']')
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
		body = nil
	}

	title1 := ExtractTitle(string(body), resp)
	finger := ExtractFinger(string(body), resp)
	var titles []string
	if title1 != "" {
		titles = append(titles, title1)
	}
	if finger != "" {
		titles = append(titles, finger)
	}
	title := strings.Join(titles, "|")
	if scanopts.OutputTitle {
		builder.WriteString(" [")
		builder.WriteString(title)
		builder.WriteRune(']')
	}

	serverHeader1 := resp.Header.Get("Server")
	serverHeader2 := resp.Header.Get("X-Powered-By")
	var serverHeaders []string
	if serverHeader1 != "" {
		serverHeaders = append(serverHeaders, serverHeader1)
	}
	if serverHeader2 != "" {
		serverHeaders = append(serverHeaders, serverHeader2)
	}
	serverHeader := strings.Join(serverHeaders, "|")

	if scanopts.OutputServerHeader {
		builder.WriteString(fmt.Sprintf(" [%s]", serverHeader))
	}

	// web socket
	isWebSocket := resp.StatusCode == 101
	if scanopts.OutputWebSocket && isWebSocket {
		builder.WriteString(" [websocket]")
	}

	return Result{
		URL:           fullURL,
		ContentLength: int(resp.ContentLength),
		StatusCode:    resp.StatusCode,
		ContentType:   resp.Header.Get("Content-Type"),
		Title:         title,
		WebServer:     serverHeader,
		str:           builder.String(),
	}
}

// Result of a scan
type Result struct {
	URL           string `json:"url"`
	Title         string `json:"title"`
	WebServer     string `json:"webserver"`
	ContentType   string `json:"content-type,omitempty"`
	ContentLength int    `json:"content-length"`
	StatusCode    int    `json:"status-code"`
	err           error
	str           string
}

// JSON the result
func (r *Result) JSON() string {
	if js, err := json.Marshal(r); err == nil {
		return string(js)
	}

	return ""
}

func GetHttpTitle(target, proc string, port int) Result {
	var scanopts = new(ScanOptions)
	scanopts.OutputTitle = true
	scanopts.OutputServerHeader = true
	result := Analyze(proc, target, port, "GET", scanopts)
	return result
}

func (r *Result) ToString(aurora aurora.Aurora) string {

	builder := &bytes.Buffer{}
	if r.err != nil {
		builder.WriteString("[")
		builder.WriteString(aurora.Red(r.err.Error()).String())
		builder.WriteString("]")
	} else {
		builder.WriteString("[")
		builder.WriteString(aurora.Green(r.StatusCode).String())
		if r.ContentLength != -1 {
			builder.WriteString("|")
			builder.WriteString(aurora.Yellow(r.ContentLength).String())
		}
		builder.WriteString("]")

		if r.Title !=""{
			builder.WriteString("[")
			builder.WriteString(aurora.Green(r.Title).String())
			builder.WriteString("]")
		}else{
			builder.WriteString("[")
			builder.WriteString(aurora.Green(r.str[:20]).String())
			builder.WriteString("]")
		}

	}

	return builder.String()
}
