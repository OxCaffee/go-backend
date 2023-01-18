<div align=center width=60%><img src="/assets/go.png"/></div>

------

##  Go基础

* [空类型nil](/docs/Go语言中的nil.md)
* [类型内嵌](/docs/Go语言中的类型内嵌.md)
* [内存对齐](docs/Go语言内存对齐.md)
* [通道channel](/docs/Go面试Channel.md)
* [结构体struct](/docs/Go面试结构体.md)
* [字典map](/docs/golang/golang面试之map.md)
* [切片slice](docs/slice源码解读.md)
* [非安全类型unsafe](/docs/unsafe.Pointer非安全类型指针.md)
* [接口interface](/docs/golang/探究Go语言中的Interface.md)

##  Go内存管理

* **GC垃圾收集**
  * [happens-before原则](/docs/Go语言HappensBefore原则.md)
  * [初步认识垃圾收集](/docs/Go语言垃圾收集器.md)
  * [On-The-Fly算法](https://lamport.azurewebsites.net/pubs/garbage.pdf)
  * [不使用STW进行垃圾收集复制](https://people.cs.umass.edu/~moss/papers/oopsla-2003-mark-copy.pdf)
  * [runtime包源码解读——mgc(1)(待完成)](/docs/golang/runtime包解读之mgc.md)
  * [runtime包源码解读——mbarrier(待完成.)]()

##  Go调度器

* [runtime包源码解读——golang程序的启动流程](/docs/golang/runtime包解读之golang程序启动流程.md)
* [runtime包源码解读——调度器的初始化schedinit](/docs/golang/runtime包解读之调度器初始化scheinit.md)
* [runtime包源码解读——调度器的入口mainPC](/docs/golang/runtime包解读之调度器入口mainPC.md)
* [runtime包源码解读——proc调度器模型](/docs/golang/runtime包解读之proc.md)

##  Go并发编程

* [并发编程之同步原语和锁机制](docs/Go语言并发同步原语和锁.md)
* [并发编程之上下文context](/docs/Go语言上下文Context.md)
* [并发编程之通道chan](/docs/Go语言通道Channel.md)
* [并发编程之sync.Mutex](/docs/sync包之mutex.md)
* [并发编程之sync.Map](/docs/sync包之map.md)
* [并发编程之sync.WaitGroup](/docs/sync包之waitGroup.md)
* [并发编程之sync.Once](/docs/sync包之once.md)

##  Go微服务

* [grpc之Interceptor拦截器](/docs/grpc之拦截器.md)
* [grpc之Server启动流程](/docs/grpc之Server启动.md)

## 场景

* [如何优雅关闭channel](/docs/golang/如何优雅地关闭channel.md)

##  数据库

* **Redis**
  * [Redis命令手册速记](/docs/redis/Redis操作手册速查.md)
  * [Redis面试题汇总](/docs/redis/redis面试题汇总.md)

##  分布式

* [分布式一致性协议](/docs/etcd/分布式一致性协议.md)
* [分布式事务](/docs/分布式事务.md)