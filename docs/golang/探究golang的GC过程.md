# 探究Golang的GC过程

## GC的行为

当GC**开始**的时候，要经历3个阶段，其中2个阶段需要`STW(Stop The World)`，这3个阶段分别为：

* `Mark Setup` ：需要`STW`
* `Concurrently Marking` ：并发标记阶段
* `Mark Termination` :需要`STW`

### Mark Setup

当GC开始的时候，Golang内核的第一件事情就是开启写屏障(`Write Barrier`)，开启写屏障的原因是：**为了保持Heap内的数据具有一致性(data integrity)，因为GC和用户协程(application goroutines)是并发运行的，如果在GC工作期间，Heap上的对象发生变化，那么就会给程序带来不可估量的后果，比如一个本不该被回收的对象被回收了，那么就会触发空指针异常** 。

为了让写屏障能够正常开启，所有的用户协程都需要停止所有的活动，这个阶段被称为`STW`，这个阶段往往时间很快，一般只有10~30ms。

**这就来了一个问题，`STW`阶段是如何让所有的goroutine停止的?** 

唯一的办法就是GC观察所有的goroutine并且等待所有的goroutine去进行一次**系统调用(system function call)** ，系统调用能够保证所有的goroutine能够在一个**安全点(safe point)**停止

