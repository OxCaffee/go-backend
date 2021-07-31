# http包源码解读——Response响应

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [Response结构体的定义](#Response)
* 3. [ReadResponse从io中解析数据并封装成Response](#ReadResponseioResponse)
* 4. [Write将Response写出到io流中](#WriteResponseio)
* 5. [transaferWriter.writeBody真正写出HTTP响应体](#transaferWriter.writeBodyHTTP)
* 6. [问题](#-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

[`net/http`]()中[`Response`]()源码并不是很多，大约为300行，主要定义了一些关于响应的读写操作。

##  2. <a name='Response'></a>Response结构体的定义

```go
type Response struct {
	Status     string // e.g. "200 OK"
	StatusCode int    // e.g. 200
	Proto      string // e.g. "HTTP/1.0"
	ProtoMajor int    // e.g. 1
	ProtoMinor int    // e.g. 0

	Header Header
	Body io.ReadCloser
	ContentLength int64
	TransferEncoding []string
	Close bool
	Trailer Header
	Request *Request
	TLS *tls.ConnectionState
}
```

可以看到基本的定义和[`Request`]()并没有太大的区别，一些字段的含义甚至是一样的，我们着重看一下[`Response`]()有哪些关键方法。

##  3. <a name='ReadResponseioResponse'></a>ReadResponse从io中解析数据并封装成Response

[`ReadResponse`]()从[`bufio.Reader`]()中读取并返回一个HTTP响应，第二个参数[`Request`]()可选地指出了此次响应对应的请求(也就是[`Request`]()中的[`Response`]()字段)。如果没有包含该字段，即该字段为[`nil`]()，**则假设此次请求是GET请求** 。

客户端在读取**响应体**完成之后必须调用[`resp.Body.Close()`]() ，也就是我们在代码中需要显式地指定关闭操作。而在服务器端，[`Response`]()在被发送出去之后将会把响应体关闭(**响应体其实就是IO读取写入器**)

```go
func ReadResponse(r *bufio.Reader, req *Request) (*Response, error) {
    // 根据 bufio.Reader来获取textproto.Reader，之后的读取操作全部交给textproto.Reader来完车
	tp := textproto.NewReader(r)
    
    // 先初始化要返回的响应resp的Request字段，标注此次响应是和哪个请求绑定的
	resp := &Response{
		Request: req,
	}

	// 解析响应的第一行，注意，此时响应的信息还是存储在textproto,Reader中的
	line, err := tp.ReadLine()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
    // 响应的第一行一般为 HTTP/1.1 200 OK，因此，若找不到' ‘，则说明此次响应格式非法
	if i := strings.IndexByte(line, ' '); i == -1 {
		return nil, badStringError("malformed HTTP response", line)
	} else {
        // 解析协议，例如HTTP/1.1
		resp.Proto = line[:i]
        // 解析状态字符串，这里的状态字符串包含了状态码和状态字，例如"200 OK"
		resp.Status = strings.TrimLeft(line[i+1:], " ")
	}
    // 解析状态码，例如200
	statusCode := resp.Status
	if i := strings.IndexByte(resp.Status, ' '); i != -1 {
		statusCode = resp.Status[:i]
	}
	if len(statusCode) != 3 {
		return nil, badStringError("malformed HTTP status code", statusCode)
	}
	resp.StatusCode, err = strconv.Atoi(statusCode)
	if err != nil || resp.StatusCode < 0 {
		return nil, badStringError("malformed HTTP status code", statusCode)
	}
	var ok bool
	if resp.ProtoMajor, resp.ProtoMinor, ok = ParseHTTPVersion(resp.Proto); !ok {
		return nil, badStringError("malformed HTTP version", resp.Proto)
	}

	// 解析响应的header
	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
    // 设置响应的header
	resp.Header = Header(mimeHeader)
    // 缓存相关，例如HTTP中的 Cache-Control: no-cache
	fixPragmaCacheControl(resp.Header)

	err = readTransfer(resp, r)
	if err != nil {
		return nil, err
	}

    // 返回响应体resp
	return resp, nil
}
```

##  4. <a name='WriteResponseio'></a>Write将Response写出到io流中

话不多说，直接上源码：

```go
func (r *Response) Write(w io.Writer) error {
	// 前面介绍到，Response其实已经封装完成，因此这里可以直接获取相应的信息，例如Status
	text := r.Status
	if text == "" {
		var ok bool
        // 如果status没有设置，则根据statusCode获取status，例如200->OK
		text, ok = statusText[r.StatusCode]
		if !ok {
			text = "status code " + strconv.Itoa(r.StatusCode)
		}
	} else {
		// 进一步优化字符串
		text = strings.TrimPrefix(text, strconv.Itoa(r.StatusCode)+" ")
	}

    // 看到没有，这里就是我们首先看到的 HTTP/1.1 200 OK 字符串了
	if _, err := fmt.Fprintf(w, "HTTP/%d.%d %03d %s\r\n", r.ProtoMajor, r.ProtoMinor, r.StatusCode, text); err != nil {
		return err
	}

	// 克隆一个resp的副本r1，这样我们在r1的基础上进行修改
	r1 := new(Response)
	*r1 = *r
    
    // 如果ContentLength=0并且body不为空，我们就要去判断是否是长度未知的
	if r1.ContentLength == 0 && r1.Body != nil {
		// Is it actually 0 length? Or just unknown?
		var buf [1]byte
		n, err := r1.Body.Read(buf[:])
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			// Reset it to a known zero reader, in case underlying one
			// is unhappy being read repeatedly.
			r1.Body = NoBody
		} else {
			r1.ContentLength = -1
			r1.Body = struct {
				io.Reader
				io.Closer
			}{
				io.MultiReader(bytes.NewReader(buf[:1]), r.Body),
				r.Body,
			}
		}
	}
	// 如果我们要发送一个非分块HTTP响应并且没有设置content-length，那么只能通过老的HTTP/1.1方式进行
    // 发送，直到到达EOF才会关闭
	if r1.ContentLength == -1 && !r1.Close && r1.ProtoAtLeast(1, 1) && !chunked(r1.TransferEncoding) && !r1.Uncompressed {
		r1.Close = true
	}

	// 处理 Body,ContentLength,Close,Trailer相关
	tw, err := newTransferWriter(r1)
	if err != nil {
		return err
	}
	err = tw.writeHeader(w, nil)
	if err != nil {
		return err
	}

	// 重新设置Header
	err = r.Header.WriteSubset(w, respExcludeHeader)
	if err != nil {
		return err
	}

	// contentLengthAlreadySent may have been already sent for
	// POST/PUT requests, even if zero length. See Issue 8180.
	contentLengthAlreadySent := tw.shouldSendContentLength()
	if r1.ContentLength == 0 && !chunked(r1.TransferEncoding) && !contentLengthAlreadySent && bodyAllowedForStatus(r.StatusCode) {
		if _, err := io.WriteString(w, "Content-Length: 0\r\n"); err != nil {
			return err
		}
	}

	// End-of-header
	if _, err := io.WriteString(w, "\r\n"); err != nil {
		return err
	}

	// 使用TransaferWriter写出body
	err = tw.writeBody(w)
	if err != nil {
		return err
	}

	// Success
	return nil
}
```

其实HTTP响应的写出交给了两个Writer来完成：

* [`bufio.Writer`]() ：这个Writer实际上是写出了HTTP 头部的一些信息，并没有写出HTTP Body
* [`transaferWriter`]() ：这个才是真正写出响应体的Writer

##  5. <a name='transaferWriter.writeBodyHTTP'></a>transaferWriter.writeBody真正写出HTTP响应体

```go
func (t *transferWriter) writeBody(w io.Writer) (err error) {
	var ncopy int64
	closed := false
	defer func() {
		if closed || t.BodyCloser == nil {
			return
		}
		if closeErr := t.BodyCloser.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	// 写响应体，首先先解除对Body的一些封装，这样能够更好地在OS层面对其进行优化，特别是涉及到os.File类型的Body信息时
	if t.Body != nil {
		var body = t.unwrapBody()
        // 如果是分块发送
		if chunked(t.TransferEncoding) {
			if bw, ok := w.(*bufio.Writer); ok && !t.IsResponse {
				w = &internal.FlushAfterChunkWriter{Writer: bw}
			}
            // 使用chunckedWriter写
			cw := internal.NewChunkedWriter(w)
			_, err = t.doBodyCopy(cw, body)
			if err == nil {
				err = cw.Close()
			}
		} else if t.ContentLength == -1 {
			dst := w
			if t.Method == "CONNECT" {
				dst = bufioFlushWriter{dst}
			}
			ncopy, err = t.doBodyCopy(dst, body)
		} else {
			ncopy, err = t.doBodyCopy(w, io.LimitReader(body, t.ContentLength))
			if err != nil {
				return err
			}
			var nextra int64
			nextra, err = t.doBodyCopy(io.Discard, body)
			ncopy += nextra
		}
		if err != nil {
			return err
		}
	}
	if t.BodyCloser != nil {
		closed = true
		if err := t.BodyCloser.Close(); err != nil {
			return err
		}
	}

	if !t.ResponseToHEAD && t.ContentLength != -1 && t.ContentLength != ncopy {
		return fmt.Errorf("http: ContentLength=%d with Body length %d",
			t.ContentLength, ncopy)
	}

	if chunked(t.TransferEncoding) {
		// 写 Trailer header
		if t.Trailer != nil {
			if err := t.Trailer.Write(w); err != nil {
				return err
			}
		}
		// 最后一个分块，emptyTrailer
		_, err = io.WriteString(w, "\r\n")
	}
	return err
}
```

##  6. <a name='-1'></a>问题

* HTTP中的Trailer字段有什么用？
* HTTP响应是如何被写出的？
* HTTP输出响应的时候使用了多少个Writer?

