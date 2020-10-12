package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

type HttpClient interface {
	SetHeader(key, value string)
	SetTimeout(timeout time.Duration)
	SetWriteTimeout(timeout time.Duration)
	SetReadTimeout(timeout time.Duration)
	Get(url string) (string, error)
	Head(url string) error
	Post(url string, body io.Reader) (string, error)
}

type Error struct {
	msg string
}

func (e *Error) Error() string {
	return e.msg
}

type HttpClientImpl struct {
}

type HttpProtocol struct {
	Method          string
	Url             string
	ProtocolVersion string
	Headers         map[string]string
	Body            string
}

func (h *HttpProtocol) toBytes() []byte {
	line := "%s %s %s\r\n" //请求行
	line = fmt.Sprintf(line, h.Method, h.Url, h.ProtocolVersion)
	headers := ""
	for k, v := range h.Headers {
		header := k + ": " + v + "\r\n"
		headers += header
	}
	headers += "\r\n"

	msg := line + headers + h.Body
	return []byte(msg)
}

type Request struct {
	Headers      map[string]string
	Method       string
	Timeout      time.Duration
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
	Url          string
	Body         io.Reader
	Port         string
}

func NewRequest() Request {
	request := Request{
		Headers: make(map[string]string),
	}
	request.SetHeader("User-Agent", "djh-go-httpClient/1.0")
	request.SetHeader("Connection", "keep-alive")
	request.SetHeader("Accept", "application/json, application/xml, text/javascript, */*")
	return request
}

func logError(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func (r *Request) doRequest() (string, error) {
	httpProtocol := HttpProtocol{
		Method:          r.Method,
		Url:             "/",
		ProtocolVersion: "HTTP/1.1",
		//Headers:         r.Headers,
		Body: "",
	}
	splitUrl := strings.Split(r.Url, "://")
	if splitUrl[0] == "http"{
		r.Port = "80"
	}else if splitUrl[0] == "https" {
		r.Port = "443"
	}else {
		return "", &Error{
			msg: "非http协议",
		}
	}
	url := strings.Split(splitUrl[1], "/")
	r.SetHeader("Host", url[0])
	portSplit := strings.Split(url[0], ":")
	if len(portSplit) > 1{
		r.Port = portSplit[1]
	}

	if len(url) > 1 {
		for i, v := range url {
			if i == 0 {
				continue
			}
			httpProtocol.Url += v + "/"
		}
	}
	if r.Body != nil {
		//todo 转body的内容

	}
	httpProtocol.Headers = r.Headers
	data := httpProtocol.toBytes()
	println(string(data))
	conn, err := net.Dial("tcp", r.Headers["Host"]+":"+r.Port)
	defer conn.Close()
	if err != nil{
		log.Fatal(err)
		return "",err
	}
	resultBytes := make([]byte, 0, 0)
	conn.Write(data)
	buf := make([]byte, 0, 1024)
	len := 0
	for true {
		read, err := conn.Read(buf)
		if read <= 0{
			break
		}
		if err != nil{
			log.Fatal(err)
			break
		}
		len += read
		BytesCombine(resultBytes,buf)
		buf = make([]byte, 0, 1024)
	}

	if err != nil{
		log.Fatal(err)
		return "",err
	}
	resp := string(resultBytes)
	println(resp)
	return resp,nil
}

func BytesCombine(pBytes ...[]byte) []byte {
	return bytes.Join(pBytes, []byte(""))
}

func (r *Request) SetHeader(key, value string) {
	r.Headers[key] = value
}

func (r *Request) Get(url string) (string, error) {
	r.Method = "GET"
	r.Url = url
	return r.doRequest()
}

func main() {
	request := NewRequest()
	resp, err := request.Get("http://www.baidu.com")
	println(resp)
	if err != nil{
		println(err.Error())
	}
}
