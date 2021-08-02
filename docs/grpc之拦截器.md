# grpc之拦截器Interceptor

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [grpc中拦截器Interceptor的定义](#grpcInterceptor)
	* 2.1. [UnaryServerInterceptor一元服务端拦截器](#UnaryServerInterceptor)
	* 2.2. [UnaryClientInterceptor一元客户端拦截器](#UnaryClientInterceptor)
	* 2.3. [StreamServerInterceptor流服务器拦截器](#StreamServerInterceptor)
	* 2.4. [StreamClientInterceptor流客户端拦截器](#StreamClientInterceptor)
* 3. [如何自定义Server端的一元Interceptor](#ServerInterceptor)
* 4. [如何注册一元Interceptor到Server端](#InterceptorServer)
* 5. [拦截器链的构建](#-1)
	* 5.1. [UnaryServerInterceptor拦截链的构建](#UnaryServerInterceptor-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

在项目开发的过程中，我们经常需要拦截器[`Interceptor`]()来完成一些操作的前置处置，例如在处理HTTP请求之前，首先需要校验HTTP的参数的合法性，在接收数据的时候，通过拦截器进行一些请求参数的解码等等，拦截器是程序设计中不可或缺的一个环节。

[`grpc`]()中同样进行了拦截器架构的相关设计，通过拦截器，可以完成对RPC调用请求或者响应的统一处理，省去了很多冗余的代码，本文着重探讨[`grpc`]()中的拦截器原理。

##  2. <a name='grpcInterceptor'></a>grpc中拦截器Interceptor的定义

[`grpc`]()定义了很多类型的拦截器，如果按照服务对象不同，可以将拦截器分为服务器端拦截器[`XxxServerInterceptor`]()和客户端拦截器[`XxxClientInterceptor`]() ，如果按照处理数据类型的不同，可以将拦截器分为普通拦截器和流式拦截器[`XxxStreamXxxxInterceptor`]() 。

###  2.1. <a name='UnaryServerInterceptor'></a>UnaryServerInterceptor一元服务端拦截器

[`UnaryServerInterceptor`]()提供了对于[`Server`]()端的[`RPC`]()调用的拦截操作，通过4个参数完成对于[`RPC`]()过程的调用，首先先看一下该拦截器的定义:

```go
type UnaryServerInterceptor func(ctx context.Context, req interface{}, info *UnaryServerInfo, handler UnaryHandler) 
		(resp interface{}, err error)
```

[`UnaryServerInterceptor`]()的本质是一个函数，需要4个参数，分别为：

* [`ctx context.Context`]() ：一个能够保存截至时间，Goroutine取消信号和其他一些全局性信息的上下文
* [`req interface{}`]() ：RPC请求参数
* [`info *UnaryServerInfo`]() ：存储该RPC方法全部信息
* [`handler UnaryHandler`]() ：**RPC方法本身，负责由[`UnaryServerInterceptor`]()调用来完成RPC方法的执行**

###  2.2. <a name='UnaryClientInterceptor'></a>UnaryClientInterceptor一元客户端拦截器

[`UnaryClientInterceptor`]()提供了对于[`Client`]()端的[`RPC`]()调用拦截操作，客户端的拦截操作相较于服务器端略显复杂。首先看一下该拦截器的定义：

```go
type UnaryClientInterceptor func(ctx context.Context, method string, req, 
          reply interface{}, cc *ClientConn, invoker UnaryInvoker, opts ...CallOption) error
```

客户端的拦截需要6个参数，分别为：

* [`ctx context.Context`]() ：上下文
* [`method string`]() ：RPC方法名称
* [`req, reply interface{}`]() ：请求和响应信息
* [`cc *ClientConn`]() ：客户端需要维持的连接指针
* [`invoker UnaryInvoker`]() ：对比于服务器端的[`UnaryHandler`]() ，也是负责RPC过程的调用的
* [`opts ...CallOption`]() ：包含了所有可用的调用参数，包括默认的[`ClientConn`]()参数

###  2.3. <a name='StreamServerInterceptor'></a>StreamServerInterceptor流服务器拦截器

```go
type StreamServerInterceptor func(srv interface{}, ss ServerStream, info *StreamServerInfo, handler StreamHandler) error
```

###  2.4. <a name='StreamClientInterceptor'></a>StreamClientInterceptor流客户端拦截器

```go
type StreamServerInterceptor func(srv interface{}, ss ServerStream, info *StreamServerInfo, handler StreamHandler) error
```

##  3. <a name='ServerInterceptor'></a>如何自定义Server端的一元Interceptor

从上面的源码可以看到[`Interceptor`]()本质并不是一个具体的结构，而是一个函数，因此拦截器准确的说应该称为是拦截函数[`Interceptor function`]() ，因此，要想定义一个拦截器，只需要完成这个拦截器本身的函数体即可，例如，我们想要在[`Server`]()端收到RPC请求的时候打印一个字符串，只需要像下面这样定义：

```go
var myInterceptor grpc.UnaryServerInterceptor
myInterceptor = func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler)(
		resp interface{}, err error) {
		// 拦截之前我们需要做什么
		fmt.Println("检测到RPC请求")
		// handler负责实现真正的RPC调用
		return handler(ctx, req)
	}
```

##  4. <a name='InterceptorServer'></a>如何注册一元Interceptor到Server端

有了自定义拦截器，但是工作还没有结束，现在这个自定义拦截器还处于游离态，并没有注册到[`Server`]()里，Golang并不支持[`Server`]()在启动的时候动态添加拦截器，事实上，大多数语言都不支持，因此，我们自定义好的拦截器是**作为Server初始化参数注册进Server端的** ：

```go
s := grpc.NewServer([]grpc.ServerOption{grpc.UnaryInterceptor(myInterceptor)}...)
```

这里用到了一个非常重要但是又非常难以理解的结构[`ServerOption`]() ，由于[`ServerOption`]()是[`Server`]()初始化中一个非常重要的参数，在这里展开又会使篇幅过长，因此具体的探究会放到介绍[RPC服务器启动流程]()文章中去介绍。在这里，我们仅仅需要了解拦截器是如何注册到[`Server`]()中的即可。

然而[`Server`]()端仅仅允许注册一个拦截器（实际上客户端也是如此），如果注册超过多于一个的拦截器，将会报错：

```go
func UnaryInterceptor(i UnaryServerInterceptor) ServerOption {
	return newFuncServerOption(func(o *serverOptions) {
        // 注册多余一个拦截器，将会报错
		if o.unaryInt != nil {
			panic("The unary server interceptor was already set and may not be reset.")
		}
		o.unaryInt = i
	})
}
```

在这种情况下，如果想要注册多个拦截器，只能将这些拦截器串联起来，构成一个拦截器链。

##  5. <a name='-1'></a>拦截器链的构建

###  5.1. <a name='UnaryServerInterceptor-1'></a>UnaryServerInterceptor拦截链的构建

构建拦截器链，其中一种方式就是利用闭包调用：[`f = inter(ctx, f1); f1 = inter(ctx, f2); ....`]() ，在代码层面，就表现为拦截器的闭包：

```go
func ChainUnaryServer(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
    // 获取要添加到Server端的拦截器数组
	n := len(interceptors)

	// 如果要添加的拦截器个数为0，那么仅需ctx和req以及handler，其中handler负责调用rpc方法
	if n == 0 {
		return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	// 如果只有一个拦截器，无需构建链，直接返回即可
	if n == 1 {
		return interceptors[0]
	}

	// 利用闭包的方式构建
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // currHandler目前表示第一个要执行的handler，也就是拦截链最外层的handler
		currHandler := handler
		// 依次遍历拦截器数组
		for i := n - 1; i > 0; i-- {
            // 向后遍历除了第一个(最外层)拦截器之外的拦截器，将每个函数包装在一个满足处理程序接口的函数中
            // 同时也是一个闭包调用，然后将每个伪处理程序作为要调用的物理程序传递给下一个外部拦截器
            // 注意，若拦截链是inter0->inter1->inter2....，那么inter(n)是先执行的
			innerHandler, i := currHandler, i
            // 白话就是：我当前的处理程序Handler的任务是调用我里层的处理程序Handler
			currHandler = func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return interceptors[i](currentCtx, currentReq, info, innerHandler)
			}
		}
		// 我只需要调用interceptor[0]，0会负责调用1，1会负责调用2.....依此类推
		return interceptors[0](ctx, req, info, currHandler)
	}
}
```

事实上我们还有一种更加容易理解的构建方式：

```go
func InterceptChain(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	// 获取拦截器链的长度
	l := len(interceptors)
	// 我们返回一个拦截器，这个拦截器就是我们的入口拦截器，从最外层递归调用最里层
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
				handler grpc.UnaryHandler) (resp interface{}, err error) {
		// 构建调用链
		chain := func(currentInterceptor grpc.UnaryServerInterceptor,
				currentHandler grpc.UnaryHandler) grpc.UnaryHandler {
			return func(ctx context.Context, req interface{}) (interface{}, error) {
				return currentInterceptor(ctx, req, info, currentHandler)
			}
		}

		// 声明一个Handler
		chainHandler := handler
		for i := l - 1; i >= 0; i-- {
			// 递归调用关系的构建
            // 拦截器(n-1)调用了handler并 
			chainHandler = chain(interceptors[i], chainHandler)
		}
		return chainHandler(ctx, req)
	}
}
```







