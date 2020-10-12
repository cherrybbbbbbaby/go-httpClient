package main

import (
	"bytes"
	"errors"
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

//HTTP协议
type HttpProtocol struct {
	Method          string
	Url             string
	ProtocolVersion string
	Headers         map[string]string
	Body            string
}

//协议转字节数组
func (h *HttpProtocol) toBytes() []byte {
	line := "%s %s %s\r\n" //请求行
	line = fmt.Sprintf(line, h.Method, h.Url, h.ProtocolVersion)
	headers := "" //header
	for k, v := range h.Headers {
		header := k + ": " + v + "\r\n"
		headers += header
	}
	headers += "\r\n"

	msg := line + headers + h.Body + "\r\n\r\n" //拼接
	return []byte(msg)
}

//HttpClient实现
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

//初始化请求对象
func NewRequest() Request {
	request := Request{
		Headers: make(map[string]string),
	}
	request.SetHeader("User-Agent", "djh-go-httpClient/1.0")
	request.SetHeader("Connection", "keep-alive")
	request.SetHeader("Accept", "application/json, application/xml, text/javascript, */*")
	return request
}

func (r *Request) doRequest() (string, error) {
	//初始化http协议
	httpProtocol := HttpProtocol{
		Method:          r.Method,
		Url:             "/",
		ProtocolVersion: "HTTP/1.1",
		//Headers:         r.Headers,
		Body: "",
	}

	//读取body
	if r.Body != nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		body := buf.String()
		httpProtocol.Body = body
	}

	var url []string
	if strings.Index(r.Url, "://") > 0 {
		//显示声名了协议 进行解析
		splitUrl := strings.Split(r.Url, "://")
		//解析协议 设置默认端口
		if splitUrl[0] == "http" {
			r.Port = "80"
		} else if splitUrl[0] == "https" {
			r.Port = "443"
		} else {
			return "", errors.New("非http协议")
		}
		url = strings.Split(splitUrl[1], "/")
	} else {
		//未声明协议 默认http协议
		url = []string{r.Url}
		r.Port = "80"
	}

	//设置header中的Host
	r.SetHeader("Host", url[0])
	portSplit := strings.Split(url[0], ":")
	if len(portSplit) > 1 {
		//解析是否声名了端口
		r.Port = portSplit[1]
	}

	//设置http协议中的的url
	if len(url) > 1 {
		for i, v := range url {
			if i == 0 {
				continue
			}
			httpProtocol.Url += v + "/"
		}
	}

	httpProtocol.Headers = r.Headers
	data := httpProtocol.toBytes()
	conn, err := net.Dial("tcp", r.Headers["Host"]+":"+r.Port)
	defer conn.Close()
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	//请求
	conn.Write(data)

	//读取响应
	buffer := new(bytes.Buffer)
	read, err := io.Copy(buffer, conn)
	println("copy :", read)
	if err != nil {
		return "", err
	}
	resp := string(buffer.Bytes())

	//解析状态 非200状态返回error
	i := strings.Index(resp, "\r\n")
	status := resp[:i]
	err = nil
	statusSplit := strings.Split(status, " ")
	if statusSplit[1] != "200" {
		err = errors.New(status)
	}
	return resp, err
}

func (r *Request) SetHeader(key, value string) {
	r.Headers[key] = value
}

func (r *Request) Get(url string) (string, error) {
	r.Method = "GET"
	r.Url = url
	return r.doRequest()
}

func (r *Request) Head(url string) error {
	r.Method = "HEAD"
	r.Url = url
	_, err := r.doRequest()
	return err
}

func (r *Request) Post(url string, body io.Reader) (string, error) {
	r.Method = "POST"
	r.Url = url
	r.Body = body
	return r.doRequest()
}

func main() {
	//request := NewRequest()
	//resp, err := request.Get("http://market.v8keji.cn/")
	//println(resp)
	//if err != nil{
	//	println(err.Error())
	//}

	request := NewRequest()
	err := request.Head("http://www.baidu.com/1")
	if err != nil {
		log.Fatal(err.Error())
	}

}
