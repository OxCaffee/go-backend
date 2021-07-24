# Go面试——Channel

<!-- vscode-markdown-toc -->
* 1. [channel的合理退出](#channel)
	* 1.1. [方式一:select语句](#:select)
	* 1.2. [方式二:WaitGroup](#:WaitGroup)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name='channel'></a>channel的合理退出

###  1.1. <a name=':select'></a>方式一:select语句

```go
func main() {
	c := make(chan int)
	done := make(chan bool)

    // go1
	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println(i)
			c <- i
		}
		done <- true
	}()

    // go2
	go func() {
		for {
			select {
			case <-c:
				continue
			case <-done:
				fmt.Println("done")
				return
			}
		}
	}()

	<-done
	fmt.Println()
}
```

上面的代码片段开辟了两个goroutine，我们分别称为go1和go2，go1负责生产10个数，并且推送到[`channel c`]()中，go2负责[`channel c`]()中取出数字，并在接收到[`done`]()信号时退出，看似没有什么问题，运行一下，结果如下：

```go
PS D:\Github\repo\Go-Backend> go run "d:\Github\repo\Go-Backend\basic\channel.go"
0
1
2
3
4
5
6
7
8
9
```

可以看到[`"done"`]()并没有被打印，为什么呢？

**当go1的最后一个数字9被推送到[`channel c`]() 中的时候，[`true`]()信号会被紧接着推送到[`channel done`]()中，此时main中的[`<-done`]()已经阻塞了很久，接收到信号之后立马恢复运行，程序退出，而此时go2才刚刚阻塞，主协程main挂掉，剩下的子协程接着挂掉** 。

###  1.2. <a name=':WaitGroup'></a>方式二:WaitGroup

```go
func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for i := 0; i < 10; i++ {
			fmt.Println(i)
		}
		wg.Done()
	}()

	wg.Wait()
}
```

