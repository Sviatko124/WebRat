package main

import (
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/widget"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/theme"
    "fmt"
    "WebRat/proxy"
    "WebRat/shared"
    "bytes"
    "net/http"
    "strings"
    "io"
    "net/url"
    "os"
    "errors"
    "time"
)
func parseRequestString(s string) (*http.Request, error) {
    fmt.Println("\n=== Starting Request Parsing ===")
    
    s = strings.ReplaceAll(s, "\r\n", "\n")
    
    headerBody := strings.Split(s, "\n\n")
    var bodyPart string
    if len(headerBody) >= 2 {
        bodyPart = strings.TrimSpace(headerBody[1])
        fmt.Printf("Found body part: '%s'\n", bodyPart)
    }

    firstLine := strings.Split(strings.Split(s, "\n")[0], " ")
    if len(firstLine) != 3 {
        return nil, fmt.Errorf("invalid request line")
    }
    
    method := firstLine[0]
    urlStr := firstLine[1]
    
    fmt.Printf("Method: %s, URL: %s\n", method, urlStr)

    var bodyReader io.Reader
    if bodyPart != "" {
        bodyReader = strings.NewReader(bodyPart)
    }
    
    req, err := http.NewRequest(method, urlStr, bodyReader)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %v", err)
    }

    headers := strings.Split(headerBody[0], "\n")[1:]
    for _, header := range headers {
        if header == "" {
            continue
        }
        parts := strings.SplitN(header, ": ", 2)
        if len(parts) == 2 {
            req.Header.Add(parts[0], parts[1])
        }
    }

    if host := req.Header.Get("Host"); host != "" {
        req.Host = host
    } else {
        req.Host = req.URL.Host
    }

    if method == "POST" && bodyPart != "" {
        req.ContentLength = int64(len(bodyPart))
        
        if req.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
            req.Body = io.NopCloser(strings.NewReader(bodyPart))
            
            formValues, err := url.ParseQuery(bodyPart)
            if err == nil {
                req.PostForm = formValues
                req.Form = formValues
                fmt.Printf("Parsed form values: %v\n", formValues)
            }
        }
    }

    fmt.Println("\n=== Final Request Details ===")
    fmt.Printf("Method: %s\n", req.Method)
    fmt.Printf("URL: %s\n", req.URL)
    fmt.Printf("Host: %s\n", req.Host)
    fmt.Printf("Content-Length: %d\n", req.ContentLength)
    fmt.Printf("Body: '%s'\n", bodyPart)
    fmt.Printf("Headers:\n")
    for k, v := range req.Header {
        fmt.Printf("%s: %v\n", k, v)
    }

    return req, nil
}

func reqToString(req *http.Request) string {
    var bodyBytes []byte
    if req.Body != nil {
        bodyBytes, _ = io.ReadAll(req.Body)
        req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
    }

    var buf strings.Builder
    
    fmt.Fprintf(&buf, "%s %s HTTP/1.1\n", req.Method, req.URL.String())
    
    for k, v := range req.Header {
        fmt.Fprintf(&buf, "%s: %s\n", k, strings.Join(v, ", "))
    }
    
    buf.WriteString("\n")
    
    if len(bodyBytes) > 0 {
        buf.Write(bodyBytes)
    }

    return buf.String()
}

func headerToString(h http.Header) string {
    var buf bytes.Buffer
    h.Write(&buf)
    return buf.String()
}

func setupInterceptorTab(requestText *widget.Entry, finishBtn *widget.Button) {
    go func() {
        for req := range shared.RequestChan {
            if req == nil {
                continue
            }
            
            reqStr := reqToString(req)
            
            fyne.DoAndWait(func() {
                requestText.SetText(reqStr)
                fmt.Println("Updated request content in UI")
            })
        }
    }()
}

func main() {
    shared.ClearBuffers()
    
    go proxy.MyServer()

    
    myApp := app.New()
    myApp.Settings().SetTheme(theme.DarkTheme())

    dark := true
    themebtn := widget.NewButton("Toggle Theme", func() {
        if dark {
            myApp.Settings().SetTheme(theme.LightTheme())
        } else {
            myApp.Settings().SetTheme(theme.DarkTheme())
        }
        dark = !dark
    })

    myWindow := myApp.NewWindow("WebRat")

    statusLabel := widget.NewLabel("Ready")

    proxyLabel := widget.NewLabel("Proxy enabled: true")
    proxyContent := container.NewVBox(
        themebtn,
        proxyLabel,
        statusLabel,
    )

    requestText := widget.NewMultiLineEntry()
    requestText.Wrapping = fyne.TextWrapWord
    requestText.SetMinRowsVisible(20)

    finishBtn := widget.NewButton("Forward Request", func() {
        modifiedText := requestText.Text
        
        requestHTTP, err := parseRequestString(modifiedText)
        if err != nil {
            fyne.DoAndWait(func() {
                statusLabel.SetText(fmt.Sprintf("Error: %v", err))
            })
            return
        }

        select {
        case shared.ResponseChan <- requestHTTP:
            fyne.DoAndWait(func() {
                statusLabel.SetText("Request forwarded successfully")
            })
        default:
            fyne.DoAndWait(func() {
                statusLabel.SetText("Failed to forward request")
            })
        }
    })

    setupInterceptorTab(requestText, finishBtn)

    tab2Content := container.NewVBox(
        widget.NewLabel("Intercepted Request Data:"),
        requestText,
        finishBtn,
    )


    intruderRequestText := widget.NewMultiLineEntry()
    intruderRequestText.SetPlaceHolder("Paste request here. Use {[^]} as wildcard...")

    intruderRequestText.Wrapping = fyne.TextWrapWord
    intruderRequestText.SetMinRowsVisible(15)

    wordlistPath := widget.NewEntry()
    wordlistPath.SetPlaceHolder("Wordlist path")
    wordlistPath.Resize(fyne.NewSize(500, 40))

    resultsText := widget.NewMultiLineEntry()
    resultsText.Wrapping = fyne.TextWrapWord
    resultsText.SetPlaceHolder("Attack results will appear here...")
    resultsText.SetMinRowsVisible(10)

    progress := widget.NewProgressBar()
    progress.Hide()

    intruderStatus := widget.NewLabel("")

    var attackRunning bool
    var cancelAttack chan bool

    var startAttackBtn *widget.Button

    loadWordlistBtn := widget.NewButton("Load Wordlist", func() {
        dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
            if err != nil {
                dialog.ShowError(err, myWindow)
                return
            }
            if reader == nil {
                return
            }
            defer reader.Close()
            
            wordlistPath.SetText(reader.URI().Path())
        }, myWindow)
    })

        startAttackBtn = widget.NewButton("Start Attack", func() {
        if attackRunning {
            if cancelAttack != nil {
                cancelAttack <- true
            }
            attackRunning = false
            startAttackBtn.SetText("Start Attack")
            progress.Hide()
            intruderStatus.SetText("Attack cancelled")
            return
        }

        if wordlistPath.Text == "" {
            dialog.ShowError(errors.New("please load a wordlist first"), myWindow)
            return
        }

        if !strings.Contains(intruderRequestText.Text, "{[^]}") {
            dialog.ShowError(errors.New("no wildcard {[^]} found in request"), myWindow)
            return
        }

        resultsText.SetText("")
        progress.SetValue(0)
        progress.Show()
        
        attackRunning = true
        startAttackBtn.SetText("Stop Attack")
        cancelAttack = make(chan bool)

        go func() {
            defer func() {
                fyne.DoAndWait(func() {
                    attackRunning = false
                    startAttackBtn.SetText("Start Attack")
                    progress.Hide()
                    intruderStatus.SetText("Attack completed")
                })
            }()

            wordlist, err := os.ReadFile(wordlistPath.Text)
            if err != nil {
                fyne.DoAndWait(func() {
                    dialog.ShowError(err, myWindow)
                })
                return
            }

            words := strings.Split(string(wordlist), "\n")
            totalWords := len(words)

            client := &http.Client{
                Timeout: 10 * time.Second,
            }

            for i, word := range words {
                select {
                case <-cancelAttack:
                    return
                default:
                    word = strings.TrimSpace(word)
                    if word == "" {
                        continue
                    }

                    currentRequest := strings.ReplaceAll(intruderRequestText.Text, "{[^]}", word)
                    
                    req, err := parseRequestString(currentRequest)
                    if err != nil {
                        fyne.DoAndWait(func() {
                            resultsText.SetText(resultsText.Text + fmt.Sprintf("[Error with %s] Failed to parse request: %v\n", word, err))
                        })
                        continue
                    }

                    resp, err := client.Do(req)
                    if err != nil {
                        fyne.DoAndWait(func() {
                            resultsText.SetText(resultsText.Text + fmt.Sprintf("[Word: %s] Error: %v\n", word, err))
                        })
                        continue
                    }

                    body, _ := io.ReadAll(resp.Body)
                    resp.Body.Close()

                    fyne.DoAndWait(func() {
                        progress.SetValue(float64(i) / float64(totalWords))
                        resultsText.SetText(resultsText.Text + fmt.Sprintf("Length: %d, [Word: %s] Status: %d\n", 
                            len(body),
                            word, 
                            resp.StatusCode,
                            
                        ))
                    })

                    time.Sleep(100 * time.Millisecond)
                }
            }
        }()
    })

    intruderContent := container.NewVBox(
        widget.NewLabel("Request Template:"),
        intruderRequestText,
        container.NewHBox(
            widget.NewLabel("Wordlist:"),
            container.NewGridWrap(fyne.NewSize(400, 36),
                wordlistPath,
            ),
            loadWordlistBtn,
        ),
        startAttackBtn,
        progress,
        intruderStatus,
        widget.NewLabel("Results:"),
        resultsText,
    )


    tabs := container.NewAppTabs(
        container.NewTabItem("Proxy", proxyContent),
        container.NewTabItem("Interceptor", tab2Content),
        container.NewTabItem("Intruder", intruderContent),

    )

    myWindow.SetContent(tabs)
    myWindow.Resize(fyne.NewSize(800, 600))
    myWindow.ShowAndRun()
}