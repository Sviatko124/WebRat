package shared

import (
    "sync"
    "net/http"
)

const (
    RequestBufferSize = 100
)

var (
    OutputHeader  http.Header
    OutputURL     string
    Mu           sync.RWMutex
    RequestChan  = make(chan *http.Request, RequestBufferSize)
    ResponseChan = make(chan *http.Request, RequestBufferSize)
)

func ClearBuffers() {
    for len(RequestChan) > 0 {
        <-RequestChan
    }
    for len(ResponseChan) > 0 {
        <-ResponseChan
    }
}