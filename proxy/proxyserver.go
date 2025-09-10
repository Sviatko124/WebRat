package proxy

import (
    "log"
    "net/http"
    "sync"
    "github.com/elazarl/goproxy"
    "WebRat/shared"
)

type ProxyServer struct {
    proxy    *goproxy.ProxyHttpServer
    port     string
    stopChan chan struct{}
    wg       sync.WaitGroup
}

func NewProxyServer(port string) *ProxyServer {
    return &ProxyServer{
        proxy:    goproxy.NewProxyHttpServer(),
        port:     port,
        stopChan: make(chan struct{}),
    }
}

func (ps *ProxyServer) Start() error {
    ps.proxy.Verbose = false

    ps.proxy.OnRequest().DoFunc(
        func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
            select {
            case <-ps.stopChan:
                return req, nil
            default:
                shared.Mu.Lock()
                shared.OutputHeader = req.Header.Clone()
                shared.OutputURL = req.URL.String()
                shared.Mu.Unlock()

                shared.RequestChan <- req.Clone(req.Context())

                select {
                case modifiedReq := <-shared.ResponseChan:
                    return modifiedReq, nil
                case <-ps.stopChan:
                    return req, nil
                }
            }
        })

    ps.wg.Add(1)
    go func() {
        defer ps.wg.Done()
        server := &http.Server{
            Addr:    ":" + ps.port,
            Handler: ps.proxy,
        }

        go func() {
            <-ps.stopChan
            server.Close()
        }()

        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Printf("Proxy server error: %v", err)
        }
    }()

    return nil
}

func (ps *ProxyServer) Stop() {
    close(ps.stopChan)
    ps.wg.Wait()
}

func MyServer() {
    proxyServer := NewProxyServer("9090")
    if err := proxyServer.Start(); err != nil {
        log.Fatal("Failed to start proxy server:", err)
    }
}