package client

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

	Schema string
	Host   string
	Param  string

	Response Response

	ResponseChan     chan Response
	TimeoutChan      chan int //0完成 1超时
	WriteTimeoutChan chan int //0完成 1超时
	ReadTimeoutChan  chan int //0完成 1超时
}

type Response struct {
	Code string
	Err  error
	Data string
}

//初始化请求对象
func NewRequest() Request {
	request := Request{
		Headers: make(map[string]string),
	}
	request.SetHeader("User-Agent", "djh-go-httpClient/1.0")
	request.SetHeader("Connection", "keep-alive")
	request.SetHeader("Accept", "application/json, application/xml, text/javascript, */*")
	request.ResponseChan = make(chan Response, 1)
	request.TimeoutChan = make(chan int, 1)
	request.WriteTimeoutChan = make(chan int, 1)
	request.ReadTimeoutChan = make(chan int, 1)
	return request
}

func (r *Request) doRequest() {
	//初始化结果
	resp := Response{
		Code: "",
		Err:  nil,
		Data: "",
	}

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

	var tempHost string
	//解析协议
	if strings.Index(r.Url, "://") > 0 {
		//显示声明了协议
		splitSchemaAndHost := strings.Split(r.Url, "://")
		schema := splitSchemaAndHost[0] //协议
		if schema == "http" {
			r.Port = "80"
		} else if schema == "https" {
			r.Port = "443"
		} else {
			resp.Err = errors.New("非http协议")
			r.ResponseChan <- resp
			return
		}
		r.Schema = schema
		tempHost = splitSchemaAndHost[1] //路径
	} else {
		//没有声明协议 默认使用http和80端口
		r.Schema = "http"
		r.Port = "80"
		tempHost = r.Url
	}
	//截取host
	var host string
	if strings.Index(tempHost, ":") > 1 {
		host = tempHost[0:strings.Index(tempHost, ":")]
	} else if strings.Index(tempHost, "/") > 1 {
		host = tempHost[0:strings.Index(tempHost, "/")]
	} else {
		host = tempHost
	}
	r.Host = host

	//截取端口
	if strings.Index(tempHost, ":") > 1 {
		//声明了端口
		var hostAndPort string
		if strings.Index(tempHost, "/") > 1 {
			hostAndPort = tempHost[0:strings.Index(tempHost, "/")]
		} else {
			hostAndPort = tempHost
		}
		_, port, err := net.SplitHostPort(hostAndPort)
		if err != nil {
			//截取端口错误
			resp.Err = err
			r.ResponseChan <- resp
			return
		}
		r.Port = port
	}

	//截取参数
	param := ""
	if strings.Index(tempHost, "/") > 1 {
		param += tempHost[strings.Index(tempHost, "/"):]
	}
	r.Param = param
	r.Headers["Host"] = r.Host
	if param != "" {
		httpProtocol.Url = param
	}

	httpProtocol.Headers = r.Headers
	data := httpProtocol.toBytes()
	address := r.Host + ":" + r.Port
	conn, err := net.Dial("tcp", address)
	if err != nil {
		resp.Err = err
		r.ResponseChan <- resp
		return
	}
	defer conn.Close()
	//请求
	r.writeTimeoutCheck() //写超时检测
	conn.Write(data)

	//读取响应
	buffer := new(bytes.Buffer)
	read, err := io.Copy(buffer, conn)
	println("copy :", read)
	if err != nil {
		resp.Err = err
		r.ResponseChan <- resp
		return
	}

	r.readTimeoutCheck() //读超时检测
	resp.Data = string(buffer.Bytes())

	//解析状态 非200状态返回error
	i := strings.Index(resp.Data, "\r\n")
	status := resp.Data[:i]
	err = nil
	statusSplit := strings.Split(status, " ")
	if statusSplit[1] != "200" {
		err = errors.New(status)
	}
	resp.Code = statusSplit[1]
	resp.Err = err
	r.Response = resp
	r.ResponseChan <- resp
	return
}

func (r *Request) readTimeoutCheck() {
	if r.ReadTimeout > 0 {
		//设置了读超时
		go func() {
			time.Sleep(r.ReadTimeout)
			r.ReadTimeoutChan <- 1
		}()
	}
}

//写超时检测
func (r *Request) writeTimeoutCheck() {
	if r.WriteTimeout > 0 {
		//设置了写超时
		go func() {
			time.Sleep(r.WriteTimeout)
			r.WriteTimeoutChan <- 1
		}()
	}
}

func (r *Request) SetHeader(key, value string) {
	r.Headers[key] = value
}

func (r *Request) Get(url string) (string, error) {
	r.Method = "GET"
	r.Url = url
	resp := r.asyncDoRequestWithTimeout()
	return resp.Data, resp.Err
}

func (r *Request) Head(url string) error {
	r.Method = "HEAD"
	r.Url = url
	resp := r.asyncDoRequestWithTimeout()
	return resp.Err
}

func (r *Request) Post(url string, body io.Reader) (string, error) {
	r.Method = "POST"
	r.Url = url
	r.Body = body
	resp := r.asyncDoRequestWithTimeout()
	return resp.Data, resp.Err
}

func (r *Request) asyncDoRequestWithTimeout() Response {
	go r.doRequest()            //执行请求
	r.timeoutCheck()            //超时检测
	resp := r.receiveResponse() //获取结果
	return resp
}

func (r *Request) timeoutCheck() {
	if r.Timeout > 0 {
		go func() {
			time.Sleep(r.Timeout)
			r.TimeoutChan <- 1
		}()
	}
}

func (r *Request) receiveResponse() Response {
	var (
		resp         Response
		timeout      int
		readTimeout  int
		writeTimeout int
	)
	// 实现超时逻辑
	select {
	case resp = <-r.ResponseChan:
		println(resp.Data)
		if resp.Err != nil {
			println(resp.Err.Error())
		}
	case timeout = <-r.TimeoutChan:
		if timeout == 1 {
			//超时
			resp.Err = errors.New("timeout")
		} else {
			resp = r.Response
		}
	case readTimeout = <-r.ReadTimeoutChan:
		if readTimeout == 1 {
			//读超时
			resp.Err = errors.New("read timeout")
		} else {
			resp = r.Response
		}
	case writeTimeout = <-r.WriteTimeoutChan:
		if writeTimeout == 1 {
			//写超时
			resp.Err = errors.New("write timeout")
		} else {
			resp = r.Response
		}

		return resp
	}
	return resp
}

func (r *Request) SetTimeout(timeout time.Duration) {
	r.Timeout = timeout
}
func (r *Request) SetWriteTimeout(timeout time.Duration) {
	r.WriteTimeout = timeout
}
func (r *Request) SetReadTimeout(timeout time.Duration) {
	r.ReadTimeout = timeout
}
