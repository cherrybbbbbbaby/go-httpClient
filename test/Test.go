package main

import (
	"go-httpClient/client"
	"strings"
	"time"
)

func main() {
	url := "www.baidu.com"
	body := "{\"name\":\"djh\"}"
	doPost(url, body)                           //带application/json的post请求
	doGet(url)                                  //get
	doHead(url)                                 //head
	doGetWithTimeout(url, time.Second*10, 0, 0) //带超时的get请求，超时<=0不作处理
}

func errCheck(err error) {
	if err != nil {
		panic(err)
	}
}

//get测试
func doGet(url string) {

	request := client.NewRequest()
	resp, err := request.Get(url)
	errCheck(err)
	println(resp)
}

//head测试
func doHead(url string) {
	request := client.NewRequest()
	err := request.Head(url)
	errCheck(err)
}

//post测试
func doPost(url string, body string) {
	request := client.NewRequest()
	request.SetHeader("Content-Type", "application/json") //set header
	resp, err := request.Post(url, strings.NewReader(body))
	errCheck(err)
	println(resp)
}

//带超时的get请求
func doGetWithTimeout(url string, timeout time.Duration, readTimeout time.Duration, writeTimeout time.Duration) {
	request := client.NewRequest()
	request.SetTimeout(timeout)
	request.SetWriteTimeout(writeTimeout)
	request.SetReadTimeout(readTimeout)
	resp, err := request.Get(url)
	errCheck(err)
	println(resp)
}
