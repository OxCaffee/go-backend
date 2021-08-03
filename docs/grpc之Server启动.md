# grpc之Server启动流程

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [一个Server都包含了什么](#Server)
	* 2.1. [serverOptions](#serverOptions)
* 3. [grpc.NewServer创建服务器的过程](#grpc.NewServer)
* 4. [RegisterXxxxServer将服务注册进Server](#RegisterXxxxServerServer)
* 5. [Server.Serve启动服务器](#Server.Serve)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

[`grpc`]()中正常启动一个[`Server`]()的流程如下：

```go
func main() {
    // <1> 先设置需要对应端口port的监听器listener
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	// <2> 模拟一个我们的拦截器
	var myInterceptor grpc.UnaryServerInterceptor
	myInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler)(
		resp interface{}, err error) {
		fmt.Println("这里是server端")
		return handler(ctx, req)
	}

	// <3> 创建一个Server，将我们创建的listener添加进这个server
	s := grpc.NewServer(
		[]grpc.ServerOption{grpc.UnaryInterceptor(myInterceptor)}...)
	// <4> 将服务注册进server
	pb.RegisterGreeterServer(s, &service{})
	log.Printf("server listening at %v", lis.Addr())
    // <5> 启动server
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
```

##  2. <a name='Server'></a>一个Server都包含了什么

服务端的实例对应的数据结构是[`Server`]()，定义在[`grpc/server.go`]()中，[`Server`]()包含了一个服务器在运行过程中的全部的配置参数信息，如下：

```go
type Server struct {
    // Server的具体参数
	opts serverOptions
	// 用到的互斥锁
	mu  sync.Mutex 
	lis map[net.Listener]bool
	// 保存所有活动的服务器传输
	conns    map[string]map[transport.ServerTransport]bool
    // 该Server是否处于启动状态
	serve    bool
    // 
	drain    bool
    // 用于连接优雅停止的信号
	cv       *sync.Cond    
    // 服务名称->服务信息
	services map[string]*serviceInfo 
	events   trace.EventLog

	quit               *grpcsync.Event
	done               *grpcsync.Event
	channelzRemoveOnce sync.Once
    // Goroutine优雅地停止
	serveWG            sync.WaitGroup 
	// channelz的唯一身份号码
	channelzID int64 // channelz unique identification number
	czData     *channelzData

	serverWorkerChannels []chan *serverWorkerData
}
```

###  2.1. <a name='serverOptions'></a>serverOptions

```go
type serverOptions struct {
	creds                 credentials.TransportCredentials
	codec                 baseCodec
	cp                    Compressor
	dc                    Decompressor
    // 当前配置有效的UnaryServerInterceptor
	unaryInt              UnaryServerInterceptor
    // 当前配置有效的StreamServerInterceptor
	streamInt             StreamServerInterceptor
    // 所有的UnaryServerInterceptor，稍后会在启动过程中组织成调用链的形式
	chainUnaryInts        []UnaryServerInterceptor
    // 所有的StreamServerInterceptor，稍后会在启动过程中组织成调用链的形式
	chainStreamInts       []StreamServerInterceptor
	inTapHandle           tap.ServerInHandle
	statsHandler          stats.Handler
	maxConcurrentStreams  uint32
	maxReceiveMessageSize int
	maxSendMessageSize    int
	unknownStreamDesc     *StreamDesc
	keepaliveParams       keepalive.ServerParameters
	keepalivePolicy       keepalive.EnforcementPolicy
	initialWindowSize     int32
	initialConnWindowSize int32
	writeBufferSize       int
	readBufferSize        int
	connectionTimeout     time.Duration
	maxHeaderListSize     *uint32
	headerTableSize       *uint32
	numServerWorkers      uint32
}
```

##  3. <a name='grpc.NewServer'></a>grpc.NewServer创建服务器的过程

如果要想创建一个[`grpc`]()服务器，需要调用这个方法，同时，可以可选的添加一些创建的参数信息，例如，如果我们想要添加一个拦截器，或者想要修改[`Server`]()的一些内部参数信息，可以通过参数传递的方式添加。我们先看一下方法定义：

```go
func NewServer(opt ...ServerOption) *Server { ... }
```

可以看到[`NewServer()`]()方法接收了若干个[`ServerOption`]() ，那么这个[`ServerOption`]()是什么？我们继续来看：

```go
type ServerOption interface {
	apply(*serverOptions)
}
```

[`ServerOption`]()是一个接口，含有一个[`apply`]()方法，初步猜测应该是将我们需要配置的参数应用到[`serverOptions`]()里面，也就是上面讲到的服务端保存配置信息的数据结构。以拦截器为例，我们自定义的拦截器和[`ServerOption`]()又有什么关系？

在[`server.go`]()中，我们找到了很多以[`ServerOption`]()作为返回参数的函数，例如：

```go
func ForceServerCodec(codec encoding.Codec) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.codec = codec
	})
}

func UnaryInterceptor(i UnaryServerInterceptor) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		if o.unaryInt != nil {
			panic("The unary server interceptor was already set and may not be reset.")
		}
		o.unaryInt = i
	})
}

func ChainUnaryInterceptor(interceptors ...UnaryServerInterceptor) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
		o.chainUnaryInts = append(o.chainUnaryInts, interceptors...)
	})
}
```

原来我们传递的一些配置参数，最终都调用了[`newFuncServerOption`]()方法，封装成了[`ServerOption`]()，之后，将这些[`ServerOption`]()应用到[`serverOptions`]()里面，仅需要调用[`apply`]()方法即可。弄懂这个，下面就来看看具体的创建服务器的过程：

```go
func NewServer(opt ...ServerOption) *Server {
	// 一些server参数是有默认值的，首先获取默认的参数信息
	opts := defaultServerOptions

	// 遍历所有的ServerOption接口
	for _, o := range opt {
		// 调用所有的ServerOption接口，调用接口的apply方法，将配置信息应用到opts中
		o.apply(&opts)
	}
	// 创建Server实例s
	s := &Server{
		lis:      make(map[net.Listener]bool),
        // 这里就将配置信息作为参数应用到server中
		opts:     opts,
		conns:    make(map[string]map[transport.ServerTransport]bool),
		services: make(map[string]*serviceInfo),
		quit:     grpcsync.NewEvent(),
		done:     grpcsync.NewEvent(),
		czData:   new(channelzData),
	}

	// 将所有的单个服务器拦截器串联成一个，因为server/client默认只添加一个拦截器
	chainUnaryServerInterceptors(s)
	// 将所有的流服务器拦截器串联成一个
	chainStreamServerInterceptors(s)
	// 设置同步锁
	s.cv = sync.NewCond(&s.mu)
	if EnableTracing {
		_, file, line, _ := runtime.Caller(1)
		s.events = trace.NewEventLog("grpc.Server", fmt.Sprintf("%s:%d", file, line))
	}

	// @todo server worker到底是什么
	if s.opts.numServerWorkers > 0 {
		s.initServerWorkers()
	}

	// channel z是否开启了数据收集
	if channelz.IsOn() {
		// 将该Server注册到channel z中
		s.channelzID = channelz.RegisterServer(&channelzServer{s}, "")
	}
	return s
}
```

##  4. <a name='RegisterXxxxServerServer'></a>RegisterXxxxServer将服务注册进Server

当我们利用[`protoc`]()根据[`.proto`]()文件生成Go业务代码的时候，生成的代码中会自动包含对应的[`RegisterXxxxServer`]()方法，例如我们的[`.proto`]()文件如下：

```protobuf
// The greeting service definition.
service Greeter {
  // Sends a greeting
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}

// The request message containing the user's name.
message HelloRequest {
  string name = 1;
}

// The response message containing the greetings
message HelloReply {
  string message = 1;
}
```

上面的[`protobuf`]()文件会生成的注册服务代码如下：

```go
// s 是grpc server， srv是 service服务实例
func RegisterGreeterServer(s grpc.ServiceRegistrar, srv GreeterServer) {
	// 由 grpc 实施注册
	s.RegisterService(&Greeter_ServiceDesc, srv)
}
```

而在内部，我们的服务[`service`]()早已被[`protoc`]()封装成了[`Xxx_ServiceDesc`]() ，这个结构保存了我们服务的全部信息，如下：

```go
var Greeter_ServiceDesc = grpc.ServiceDesc{
    // service名称，默认前面加上包名
	ServiceName: "helloworld.Greeter",
    // 具体的service实例
	HandlerType: (*GreeterServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SayHello",
			Handler:    _Greeter_SayHello_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
    // 定义该service的proto文件
	Metadata: "examples/helloworld/helloworld/helloworld.proto",
}
```

不同的[`RegisterXxxxServer`]()的注册行为都会移交给[`grpc.ServerRegistrar.RegisterService`]()  ：

```go
func (s *Server) RegisterService(sd *ServiceDesc, ss interface{}) {
	if ss != nil {
		// 获得 ServiceDesc.HandlerType 类型
		ht := reflect.TypeOf(sd.HandlerType).Elem()
		// 获取service impl类型
		st := reflect.TypeOf(ss)
		// 检查 service impl是否实现了 ServiceDesc.HandlerType，如果没有，报错
		if !st.Implements(ht) {
			logger.Fatalf("grpc: Server.RegisterService found the handler of type %v that does not satisfy %v", st, ht)
		}
	}
	// 继续调用register方法
	s.register(sd, ss)
}
```

上面又调用了[`Server.register()`]()函数：

```go
// sd是描述的是具体service的method和handler，ss是要传递给service handler的service 的具体实现
func (s *Server) register(sd *ServiceDesc, ss interface{}) {
	// 加锁
	s.mu.Lock()
	// 最后解锁
	defer s.mu.Unlock()
	s.printf("RegisterService(%q)", sd.ServiceName)
	// 注册一定发生在服务器启动之前，如果先启动再注册就会报错
	if s.serve {
		logger.Fatalf("grpc: Server.RegisterService after Server.Serve for %q", sd.ServiceName)
	}
	// 如果服务已经被注册，那么也会报错，不允许重复注册服务
	if _, ok := s.services[sd.ServiceName]; ok {
		logger.Fatalf("grpc: Server.RegisterService found duplicate service registration for %q", sd.ServiceName)
	}
	// 创建出要注册的服务的实例
	info := &serviceInfo{
		serviceImpl: ss,
		methods:     make(map[string]*MethodDesc),
		streams:     make(map[string]*StreamDesc),
		mdata:       sd.Metadata,
	}
	// 依次获取要注册的服务的method
	for i := range sd.Methods {
		d := &sd.Methods[i]
		info.methods[d.MethodName] = d
	}
	// 依次获取要注册的服务的流信息
	for i := range sd.Streams {
		d := &sd.Streams[i]
		info.streams[d.StreamName] = d
	}
	// 将<serviceName, serviceInfo>存储在字典当中
	s.services[sd.ServiceName] = info
}
```

##  5. <a name='Server.Serve'></a>Server.Serve启动服务器

```go
func (s *Server) Serve(lis net.Listener) error {
	s.mu.Lock()
	s.printf("serving")
	s.serve = true
    // <1> 检查监听器，server启动必须需要监听器
	if s.lis == nil {
		// Serve called after Stop or GracefulStop.
		s.mu.Unlock()
		lis.Close()
		return ErrServerStopped
	}

	s.serveWG.Add(1)
	defer func() {
		s.serveWG.Done()
		if s.quit.HasFired() {
			// 优雅地停止服务器，等到Server完成
            // 当Server.Stop调用的时候，会执行s.done.Fire()发送一个信号
			<-s.done.Done()
		}
	}()

	ls := &listenSocket{Listener: lis}
	s.lis[ls] = true

    // 是否开启数据收集
	if channelz.IsOn() {
		ls.channelzID = channelz.RegisterListenSocket(ls, s.channelzID, lis.Addr().String())
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if s.lis != nil && s.lis[ls] {
			ls.Close()
			delete(s.lis, ls)
		}
		s.mu.Unlock()
	}()

	var tempDelay time.Duration // how long to sleep on accept failure

	for {
        // 持续监听
		rawConn, err := lis.Accept()
		if err != nil {
			if ne, ok := err.(interface {
				Temporary() bool
			}); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				s.mu.Lock()
				s.printf("Accept error: %v; retrying in %v", err, tempDelay)
				s.mu.Unlock()
				timer := time.NewTimer(tempDelay)
				select {
				case <-timer.C:
				case <-s.quit.Done():
					timer.Stop()
					return nil
				}
				continue
			}
			s.mu.Lock()
			s.printf("done serving; Accept = %v", err)
			s.mu.Unlock()

			if s.quit.HasFired() {
				return nil
			}
			return err
		}
		tempDelay = 0
		s.serveWG.Add(1)
		go func() {
			s.handleRawConn(lis.Addr().String(), rawConn)
			s.serveWG.Done()
		}()
	}
}
```









