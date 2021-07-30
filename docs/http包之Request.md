# net/http包源码解读——Request

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [Request结构体的定义](#Request)
	* 2.1. [请求的URL](#URL)
	* 2.2. [请求首部Header](#Header)
	* 2.3. [请求主体Body](#Body)
	* 2.4. [表单数据](#-1)
		* 2.4.1. [Form字段](#Form)
		* 2.4.2. [PostForm字段](#PostForm)
		* 2.4.3. [MultipartForm字段](#MultipartForm)
	* 2.5. [Cancel字段](#Cancel)
	* 2.6. [Context字段](#Context)
* 3. [Form相关函数](#Form-1)
	* 3.1. [ParseForm](#ParseForm)
	* 3.2. [ParseMultipartForm](#ParseMultipartForm)
	* 3.3. [FormValue](#FormValue)
	* 3.4. [PostFormValue](#PostFormValue)
	* 3.5. [FormFile](#FormFile)
	* 3.6. [总结](#-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

* [`Request`]()相关的代码位于[`net/http/request.go`]()路径下

##  2. <a name='Request'></a>Request结构体的定义

[`Request`]()是十分重要的一个数据结构，它表示服务器要接收或者客户端要发送的HTTP请求信息，该结构体实际上是对HTTP请求信息的封装，源码如下：

```go
type Request struct {

	// Method 指明了HTTP方法，有{GET, POST, PUT和其他的}，当为空字符串的时候表明
	// 是 GET 方法， HTTP客户端并不支持 CONNECT 方法
	Method string

	// URL 表示对服务器请求的URI或者对客户端请求的 URL
	//
	// 对于服务端请求，URL 是从 RequestURI 中存储的 URI 解析而成
	// 对于大多数的请求来说，Path和RawQuery以外的字段将会是空值
	//
	// 对于客户端请求，URL的Host指定要连接到的服务器，而Request的Host
	// 字段可选地指定要在HTTP请求中发送的Host头值。
	URL *url.URL

	// 当前请求的版本号
	//
	// 版本号只能为 HTTP/1.0 或者 HTTP/2.0
	Proto      string // "HTTP/1.0"
	ProtoMajor int    // 1
	ProtoMinor int    // 0

	// Header 包含了请求中的头部字段
	// 如果一个服务器接收到了如下的请求
	//
	// 	Host: example.com
	//	accept-encoding: gzip, deflate
	//	Accept-Language: en-us
	//	fOO: Bar
	//	foo: two
	//
	// 那么将会被组装成如下的形式
	//
	// Header = map[string][]string{
	//	"Accept-Encoding": {"gzip, deflate"},
	//	"Accept-Language": {"en-us"},
	//	"Foo": {"Bar", "two"},
	//	}
	//
	// 可以看到，Header是以map的形式存储的
	// 如果header中包含Host头部，那么将会被设置到Request.Host字段中
	// 并且将会从Header中删除
	Header Header

	// Body 是请求的data数据承载的部分
	//
	// 客户端请求的Body若为nil则表示没有Body，Request的Transport负责
	// Body的Close
	//
	// 服务器端的请求的Body永远不为nil，但是会返回EOF
	// Server负责关闭Request Body. 而Handler.ServeHTTP不负责Body的关闭
	//
	// Body 必须允许Read与Close同时调用，特别地，Close会结束掉Read的阻塞状态
	Body io.ReadCloser

	// GetBody 定义了一个可选的函数来返回Body的副本
	// 使用场景：当重定向需要多次读取Body的时候，常常出现在客户端请求
	// 使用 GetBody 仍然需要设置 Body
	//
	// 服务器请求不使用这个方法
	GetBody func() (io.ReadCloser, error)

	// ContentLength 表示Body的长度，如果值为-1表示长度未知
	// 对于客户端请求，如果 ContentLength为0并且Body不为nil，则也是长度未知
	ContentLength int64

	TransferEncoding []string

	// Close 对于客户端和服务器端有不同的含义
	// 对于服务器端，Close表示响应完请求之后是否关闭连接，
	// 对于客户端，Close表示发送完请求并且得到响应之后是否关闭连接，
	Close bool

	// 对于服务器请求，Host指定搜索URL的主机。对于HTTP/1(每个RFC 7230，章节5.4)，
	// 这要么是“主机”头的值，要么是URL本身给出的主机名。对于HTTP/2，它是“:authority”
	// 伪报头字段的值。它的形式可能是“host:port”。对于国际域名，Host可以使用Punycode
	// 或Unicode格式。如果需要，可以使用golang.org/x/net/idna将其转换为任意一种格式。
	// 为了防止DNS重新绑定攻击，服务器处理程序应该验证Host报头具有处理程序认为自己具有
	// 权威性的值。T
	Host string

	// Form包含解析过的表单数据，包括URL字段的查询参数和PATCH、POST或PUT表单数据。
	// 此字段仅在调用ParseForm后可用。HTTP客户端忽略Form而使用Body。
	Form url.Values

	//PostForm包含来自PATCH、POST或PUT主体参数的解析表单数据。
	//此字段仅在调用ParseForm后可用。HTTP客户端忽略PostForm而使用Body。
	PostForm url.Values

	//MultipartForm是解析的多部分表单，包括文件上传。该字段仅在调用parsemmultipartform
	//后可用。HTTP客户端忽略MultipartForm而使用Body。
	MultipartForm *multipart.Form

	Trailer Header

	//RemoteAddr允许HTTP服务器和其他软件记录发送请求的网络地址，通常用于日志记录。
	//该字段不是由ReadRequest填充的，也没有定义格式。这个包中的HTTP服务器在调用处
	//理程序之前将RemoteAddr设置为“IP:port”地址。此字段被HTTP客户端忽略。
	RemoteAddr string

	//RequestURI是请求行(RFC 7230，章节3.1.1)中未经修改的请求目标，由客户端发送给
	//服务器。通常应该使用URL字段。在HTTP客户端请求中设置此字段是错误的。
	// ！！！！！！也就是说，URI只能存在与对服务器的请求之中
	RequestURI string

	// TLS允许HTTP服务器和其他软件记录关于接收请求的TLS连接的信息。该字段不是由
	//ReadRequest填充的。这个包中的HTTP服务器在调用处理程序之前为启用tls的连接
	//设置字段;否则它会让字段为nil。此字段被HTTP客户端忽略
	TLS *tls.ConnectionState

	//Cancel是一个可选通道，它的关闭表明应该将客户端请求视为已取消。不是所有的RoundTripper实现都支持Cancel。
	//对于服务器请求，此字段不适用。
	//已弃用:用NewRequestWithContext来设置请求的上下文。如果Request的Cancel字段和上下文都被设置了，
	//那么Cancel是否被采用是未定义的。
	Cancel <-chan struct{}

	// Response是导致创建此请求的重定向响应。此字段仅在客户端重定向期间填充。
	Response *Response

	// CTX要么是客户端上下文，要么是服务器上下文。它只能通过使用WithContext复制整个请求来修改。
	//不导出它是为了防止人们错误地使用上下文，以及改变同一请求的呼叫者所持有的上下文。
	// 注意这里的ctx是小写，也就是没有导出，是私有字段
	ctx context.Context
}
```

###  2.1. <a name='URL'></a>请求的URL

[`Request`]()结构中的URL字段用于表示请求行中包含的URL(**注意，URL既可以用于客户端也可以用户服务器端，而URI只能用于服务器端，用于客户端会出现异常**) 。这个字段是一个指向[`url.URL`]()的指针，我们看看[`url.URL`]()是如何定义的：

```go
type URL struct {
	Scheme      string
	Opaque      string    // encoded opaque data
	User        *Userinfo // username and password information
	Host        string    // host or host:port
	Path        string    // path (relative paths may omit leading slash)
	RawPath     string    // encoded path hint (see EscapedPath method)
	ForceQuery  bool      // append a query ('?') even if RawQuery is empty
	RawQuery    string    // encoded query values, without '?'
	Fragment    string    // fragment for references, without '#'
	RawFragment string    // encoded fragment hint (see EscapedFragment method)
}
```

URL的一般格式为[`scheme://[userinfo@]host/path[?query][#fragment]`]() ，那些在[`scheme`]()后不带斜线[`//`]()的部分会被解释为：[`scheme:opaque[?query][#fragment]`]() 。

在开发中，我们常常会让客户端通过URL的查询参数向服务器传递信息，而URL中的[`RawQuery`]()字段记录的就是客户端向服务器传递的查询参数字符串。例如[`http://example.com/post?a=1&b=2`]()会将[`RawQuery`]()字段的值设置为[`a=1&b=2`]() 。虽然通过语法分析可以将前面的形式解析为**键值对**的形式，但是在这种情况下直接使用[`Request`]()中的[`Form`]()字段来获取这些键值对会方便一点。

###  2.2. <a name='Header'></a>请求首部Header

请求和响应的首部都是用[`Header`]()进行描述，在Go语言中，[`Header`]()的实现形式实际上是[`map[string][]string`]()切片映射的形式。**但是需要注意的是，如果[`Header`]()中含有Host字段，那么[`Request`]()会将Header中的Host剔除，直接添加进[`Request`]()的Host字段中** 。下面是[`Header`]()的定义形式：

```go
type Header map[string][]string
```

###  2.3. <a name='Body'></a>请求主体Body

请求主体Body并没有被设置为诸如[`string`]()或者其他简单的字面量的形式，而是设置为[`io.ReadCloser`]()的形式。源码如下：

```go
	// Body 是请求的data数据承载的部分
	//
	// 客户端请求的Body若为nil则表示没有Body，Request的Transport负责
	// Body的Close
	//
	// 服务器端的请求的Body永远不为nil，但是会返回EOF
	// Server负责关闭Request Body. 而Handler.ServeHTTP不负责Body的关闭
	//
	// Body 必须允许Read与Close同时调用，特别地，Close会结束掉Read的阻塞状态
	Body io.ReadCloser
```

而[`io.ReadCloser`]()是一个结构体，分别定义了[`Reader`]()和[`Closer`]() 。[`Reader`]()含有[`Read`]()方法，该方法接收一个字节切片作为输入，并且在执行之后返回被读取内容的字节数以及一个可选的错误作为结果。而[`Closer`]()接口拥有[`Close`]()方法，这个方法不接受任何参数但是会在出错的时候返回一个错误。

###  2.4. <a name='-1'></a>表单数据

用户在表单中输入的数据会以键值对的形式记录在请求的主体之中，然后以HTTP POST请求的形式发送到服务器当中。因为服务器在接收到这些数据的时候会进行语法分析，从而提取出数据中记录的键值对，因此我们还需要知道这些键值对在请求主体中是如何格式化的。

HTML表单的内容类型(content-type)决定了POST请求在发送键值对时将使用何种格式，其中HTML表单的内容类型是由表单[`encrypt`]()属性指定的：

```html
<form action="/process" method="post" encrypt="application/x-www-form-urlencoded">
    <input type="text" name="first_name"/>
    <input type="text" name="last_name"/>
    <input type="submmit"/>
</form>
```

[`encrypt`]()属性的默认值为[`application/x-www-form-urlencoded`]()。浏览器至少需要支持[`application/x-www-form-urlencoded`]()和[`multipart/form-data`]()这两种编码方式，除了这两种编码方式，HTML5还支持[`text/plain`]()编码方式。

* 如果我们把[`encrypt`]()属性设置为[`application/x-www-form-urlencoded`]() ，那么浏览器HTML将会把表单中的数据**编码为一个长长的查询字符串的形式** ，其中不同的键值对用[`&`]()分隔
* 如果我们把[`encrypt`]()属性设置为[`multipart/form-data`]() ，那么表单中的数据会被转换成一个[`MIME`]()报文，表单中的每一个键值对都构成了这个报文的一部分，并且每个键值对都带有它们各自的内容类型和内容配置

实际上，要想提交表单数据，GET和POST方法均可。只不过在GET方式下，请求并没有Body，因此表单的数据会以长查询字符串的形式传输。

下面我们继续回到[`net/http`]()源码部分，看看go是如何处理的。

####  2.4.1. <a name='Form'></a>Form字段

```go
// Form包含解析过的表单数据，包括URL字段的查询参数和PATCH、POST或PUT表单数据。
// 此字段仅在调用ParseForm后可用。HTTP客户端忽略Form而使用Body。
Form url.Values
```

go中的[`From`]()字段是以[`url.Value`]()的形式表示的， 其数据结构和上面的[`Header`]()一致，都是[`type Values map[string][]string`]()的形式。通过[`net/http`]()提供的方法，我们可以很容易地将[`Request`]()中的数据提取到[`From, PostForm，MultipartForm`]()等字段中，具体的提取方法会在后面讨论。

####  2.4.2. <a name='PostForm'></a>PostForm字段

[`PostForm`]()字段仅仅支持[`application/x-www-form-urlencoded`]()编码

####  2.4.3. <a name='MultipartForm'></a>MultipartForm字段

[`MultipartForm`]()字段相对应的是[`multipart/form-data`]()编码的表单数据，并且该字段仅仅表示表单键值对而不表示URL键值对，同时[`MultipartForm`]()字段的值不是一个映射，而是**一个包含了两个映射的结构** ，源码如下：

```go
type Form struct {
	Value map[string][]string		// 表单中的键值对
	File  map[string][]*FileHeader	// 文件上传相关数据
}
```

###  2.5. <a name='Cancel'></a>Cancel字段

[`Cancel`]()是一个可选通道，它的关闭表明应该将客户端请求视为已取消。不是所有的[`RoundTripper`]()实现都支持[`Cancel`]()。对于服务器请求，此字段不适用。

###  2.6. <a name='Context'></a>Context字段

[`Context`]()要么是客户端上下文，要么是服务器上下文。它只能通过使用[`WithContext`]()复制整个请求来修改。注意，该字段并没有导出。因此是结构体的私有字段。

##  3. <a name='Form-1'></a>Form相关函数

###  3.1. <a name='ParseForm'></a>ParseForm

[`ParseForm`]()负责填充[`Request.Form`]()和[`Request.PostForm`]()字段。解析的键值对来源为**URL和表单** 。在[`Request.Form`]()中，请求体Body中的参数优先于URL查询字符串。

* 对于POST, PUT, PATCH方法，它除了URL还读取Body。
* 对于其他HTTP方法，当[`Content-Type`]()不为[`application/x-www-form-urlencoded`]()时，请求体Body将不会被读取。

```go
func (r *Request) ParseForm() error {
	var err error
	if r.PostForm == nil {
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			r.PostForm, err = parsePostForm(r)
		}
		if r.PostForm == nil {
			r.PostForm = make(url.Values)
		}
	}
	if r.Form == nil {
		if len(r.PostForm) > 0 {
			r.Form = make(url.Values)
			copyValues(r.Form, r.PostForm)
		}
		var newValues url.Values
		if r.URL != nil {
			var e error
			newValues, e = url.ParseQuery(r.URL.RawQuery)
			if err == nil {
				err = e
			}
		}
		if newValues == nil {
			newValues = make(url.Values)
		}
		if r.Form == nil {
			r.Form = newValues
		} else {
			copyValues(r.Form, newValues)
		}
	}
	return err
}
```

###  3.2. <a name='ParseMultipartForm'></a>ParseMultipartForm

```go
func (r *Request) ParseMultipartForm(maxMemory int64) error {
	if r.MultipartForm == multipartByReader {
		return errors.New("http: multipart handled by MultipartReader")
	}
	var parseFormErr error
	if r.Form == nil {
		// Let errors in ParseForm fall through, and just
		// return it at the end.
        // 这里调用了ParseForm()
		parseFormErr = r.ParseForm()
	}
	if r.MultipartForm != nil {
		return nil
	}

	mr, err := r.multipartReader(false)
	if err != nil {
		return err
	}

	f, err := mr.ReadForm(maxMemory)
	if err != nil {
		return err
	}

	if r.PostForm == nil {
		r.PostForm = make(url.Values)
	}
	for k, v := range f.Value {
		r.Form[k] = append(r.Form[k], v...)
		// r.PostForm should also be populated. See Issue 9305.
		r.PostForm[k] = append(r.PostForm[k], v...)
	}

	r.MultipartForm = f

	return parseFormErr
}
```

###  3.3. <a name='FormValue'></a>FormValue

```go
func (r *Request) FormValue(key string) string {
	if r.Form == nil {
		r.ParseMultipartForm(defaultMaxMemory)
	}
	if vs := r.Form[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}
```

###  3.4. <a name='PostFormValue'></a>PostFormValue

```go
func (r *Request) PostFormValue(key string) string {
	if r.PostForm == nil {
		r.ParseMultipartForm(defaultMaxMemory)
	}
	if vs := r.PostForm[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}
```

###  3.5. <a name='FormFile'></a>FormFile

```go
func (r *Request) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	if r.MultipartForm == multipartByReader {
		return nil, nil, errors.New("http: multipart handled by MultipartReader")
	}
	if r.MultipartForm == nil {
		err := r.ParseMultipartForm(defaultMaxMemory)
		if err != nil {
			return nil, nil, err
		}
	}
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		if fhs := r.MultipartForm.File[key]; len(fhs) > 0 {
			f, err := fhs[0].Open()
			return f, fhs[0], err
		}
	}
	return nil, nil, ErrMissingFile
}
```

###  3.6. <a name='-1'></a>总结

|     字段      | 需要调用的方法或者字段 | 键值对来源 |  内容编码类型  |
| :-----------: | :--------------------: | :--------: | :------------: |
|     Form      |      ParseForm()       | URL，表单  | URL，Multipart |
|   PostForm    |        Form字段        |    表单    |      URL       |
| MultipartForm |  ParseMultipartForm()  |    表单    |   Multipart    |
|   FormValue   |          N/A           | URL，表单  |      URL       |
| PostFormValue |          N/A           |    表单    |      URL       |

