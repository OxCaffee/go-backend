# http包源码解读——Client客户端

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [Client结构体定义](#Client)
* 3. [Client#send发送请求](#Clientsend)
* 4. [Client.Get](#Client.Get)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

在[`http`]()包中，[`Client`]()代表了一个HTTP客户端，当[`Client`]()初始化参数全为0值的时候，就是默认实现[`DefaultClient`]() ，并且这个默认实现可以直接使用。[`Client`]() 可以单独的发送一次HTTP请求，获取相应的响应和错误信息。

##  2. <a name='Client'></a>Client结构体定义

```go
type Client struct {
	// Transport 指明了HTTP请求被创建的机制
    // 如果是nil，DefaultTransport将会被使用
	Transport RoundTripper

	// CheckRedirect 指定重定向的处理策略，CheckRedirect不为nil，则客户端会在HTTP
    // 重定向之前调用它，参数req和via是即将发出的请求和已经发出的请求，默认最老的先发出
    // 如何CheckRedirect为空，客户端会采取默认策略，即连续10次请求之后停止请求
	CheckRedirect func(req *Request, via []*Request) error

	// Jar 是专门为cookie服务
	//
	// Jar负责将cookie插入到每一个出战请求当中，并且会根据入站的请求更新cookie
    // Jar会轮询每一次客户端的重定向
	//
	// 如果Jar为nil，cookie仅会在请求中显式指定cookie的情况下被发送出去
	Jar CookieJar

	// 超时的内容包括请求，重定向和读取响应体等一系列操作的时间叠加
    // timeout为0表示没有超时限制
	Timeout time.Duration
}
```

##  3. <a name='Clientsend'></a>Client#send发送请求

```go
func (c *Client) send(req *Request, deadline time.Time) (resp *Response, didTimeout func() bool, err error) {
    // 首先检查Jar，判断是否能用Jar去处理Cookie
	if c.Jar != nil {
        // 为对应的url的req添加cookie
		for _, cookie := range c.Jar.Cookies(req.URL) {
			req.AddCookie(cookie)
		}
	}
    // 执行另外一个send
	resp, didTimeout, err = send(req, c.transport(), deadline)
	if err != nil {
		return nil, didTimeout, err
	}
	if c.Jar != nil {
        // 获取resp中的cookie
		if rc := resp.Cookies(); len(rc) > 0 {
            // 将该cookie添加到对应的url中
			c.Jar.SetCookies(req.URL, rc)
		}
	}
	return resp, nil, nil
}
```

```go
func send(ireq *Request, rt RoundTripper, deadline time.Time) (resp *Response, didTimeout func() bool, err error) {
	req := ireq // 使用另外一个指针指向ireq

    // 没有RoundTripper，也就无法发送req
	if rt == nil {
		req.closeBody()
		return nil, alwaysFalse, errors.New("http: no Client.Transport or DefaultTransport")
	}

    // 请求的url不能为nil
	if req.URL == nil {
		req.closeBody()
		return nil, alwaysFalse, errors.New("http: nil Request.URL")
	}

    // 请求的uri不能为nil
	if req.RequestURI != "" {
		req.closeBody()
		return nil, alwaysFalse, errors.New("http: Request.RequestURI can't be set in client requests")
	}

	// 浅拷贝ireq
	forkReq := func() {
		if ireq == req {
			req = new(Request)
			*req = *ireq // shallow clone
		}
	}

	// Most the callers of send (Get, Post, et al) don't need
	// Headers, leaving it uninitialized. We guarantee to the
	// Transport that this has been initialized, though.
	if req.Header == nil {
		forkReq()
		req.Header = make(Header)
	}

	if u := req.URL.User; u != nil && req.Header.Get("Authorization") == "" {
		username := u.Username()
		password, _ := u.Password()
		forkReq()
		req.Header = cloneOrMakeHeader(ireq.Header)
		req.Header.Set("Authorization", "Basic "+basicAuth(username, password))
	}

	if !deadline.IsZero() {
		forkReq()
	}
	stopTimer, didTimeout := setRequestCancel(req, rt, deadline)

    // 开启http事务，发送http请求并获取响应
	resp, err = rt.RoundTrip(req)
	if err != nil {
        // 响应回一个错误，停止计时器
		stopTimer()
		if resp != nil {
			log.Printf("RoundTripper returned a response & error; ignoring response")
		}
		if tlsErr, ok := err.(tls.RecordHeaderError); ok {
			// If we get a bad TLS record header, check to see if the
			// response looks like HTTP and give a more helpful error.
			// See golang.org/issue/11111.
			if string(tlsErr.RecordHeader[:]) == "HTTP/" {
				err = errors.New("http: server gave HTTP response to HTTPS client")
			}
		}
		return nil, didTimeout, err
	}
    // 无响应，即resp为nil
	if resp == nil {
		return nil, didTimeout, fmt.Errorf("http: RoundTripper implementation (%T) returned a nil *Response with a nil error", rt)
	}
	if resp.Body == nil {
		if resp.ContentLength > 0 && req.Method != "HEAD" {
			return nil, didTimeout, fmt.Errorf("http: RoundTripper implementation (%T) returned a *Response with content length %d but a nil Body", rt, resp.ContentLength)
		}
		resp.Body = io.NopCloser(strings.NewReader(""))
	}
	if !deadline.IsZero() {
		resp.Body = &cancelTimerBody{
			stop:          stopTimer,
			rc:            resp.Body,
			reqDidTimeout: didTimeout,
		}
	}
	return resp, nil, nil
}
```

##  4. <a name='Client.Get'></a>Client.Get

```go
func (c *Client) Get(url string) (resp *Response, err error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
    // 内部调用了Client.Do
	return c.Do(req)
}
```

```go
func (c *Client) Do(req *Request) (*Response, error) {
    // 又是一个调用
	return c.do(req)
}
```

```go
func (c *Client) do(req *Request) (retres *Response, reterr error) {
	if testHookClientDoResult != nil {
		defer func() { testHookClientDoResult(retres, reterr) }()
	}
    // 请求url为nil，直接关闭req.Body并返回错误
	if req.URL == nil {
		req.closeBody()
		return nil, &url.Error{
			Op:  urlErrorOp(req.Method),
			Err: errors.New("http: nil Request.URL"),
		}
	}

	var (
		deadline      = c.deadline()
		reqs          []*Request
		resp          *Response
		copyHeaders   = c.makeHeadersCopier(req)
		reqBodyClosed = false // 标注我们是否已经关闭req.Body

		// 重定向行为
		redirectMethod string
		includeBody    bool
	)
	uerr := func(err error) error {
		// body有可能已经被Client.send关闭了
		if !reqBodyClosed {
			req.closeBody()
		}
		var urlStr string
		if resp != nil && resp.Request != nil {
			urlStr = stripPassword(resp.Request.URL)
		} else {
			urlStr = stripPassword(req.URL)
		}
		return &url.Error{
			Op:  urlErrorOp(reqs[0].Method),
			URL: urlStr,
			Err: err,
		}
	}
	for {
		// 处理请求的下一条next hop
		if len(reqs) > 0 {
			loc := resp.Header.Get("Location")
			if loc == "" {
				resp.closeBody()
				return nil, uerr(fmt.Errorf("%d response missing Location header", resp.StatusCode))
			}
			u, err := req.URL.Parse(loc)
			if err != nil {
				resp.closeBody()
				return nil, uerr(fmt.Errorf("failed to parse Location header %q: %v", loc, err))
			}
			host := ""
			if req.Host != "" && req.Host != req.URL.Host {
				if u, _ := url.Parse(loc); u != nil && !u.IsAbs() {
					host = req.Host
				}
			}
			ireq := reqs[0]
			req = &Request{
				Method:   redirectMethod,
				Response: resp,
				URL:      u,
				Header:   make(Header),
				Host:     host,
				Cancel:   ireq.Cancel,
				ctx:      ireq.ctx,
			}
			if includeBody && ireq.GetBody != nil {
				req.Body, err = ireq.GetBody()
				if err != nil {
					resp.closeBody()
					return nil, uerr(err)
				}
				req.ContentLength = ireq.ContentLength
			}
			
            // 复制请求头
			copyHeaders(req)

			// 如果不是https到http，从最近请求的url中获取并添加Referer
			if ref := refererForURL(reqs[len(reqs)-1].URL, req.URL); ref != "" {
				req.Header.Set("Referer", ref)
			}
			err = c.checkRedirect(req, reqs)

			if err == ErrUseLastResponse {
				return resp, nil
			}

			const maxBodySlurpSize = 2 << 10
			if resp.ContentLength == -1 || resp.ContentLength <= maxBodySlurpSize {
				io.CopyN(io.Discard, resp.Body, maxBodySlurpSize)
			}
			resp.Body.Close()

			if err != nil {
				ue := uerr(err)
				ue.(*url.Error).URL = loc
				return resp, ue
			}
		}

		reqs = append(reqs, req)
		var err error
		var didTimeout func() bool
		if resp, didTimeout, err = c.send(req, deadline); err != nil {
			// Client.send一定会关闭body
			reqBodyClosed = true
			if !deadline.IsZero() && didTimeout() {
				err = &httpError{
					err:     err.Error() + " (Client.Timeout exceeded while awaiting headers)",
					timeout: true,
				}
			}
			return nil, uerr(err)
		}

		var shouldRedirect bool
		redirectMethod, shouldRedirect, includeBody = redirectBehavior(req.Method, resp, reqs[0])
		if !shouldRedirect {
			return resp, nil
		}

		req.closeBody()
	}
}
```

