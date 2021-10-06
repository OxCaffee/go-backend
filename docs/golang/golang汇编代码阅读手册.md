# Go语言汇编代码阅读手册

<!-- vscode-markdown-toc -->

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

## 伪汇编

Go 编译器会输出一种抽象可移植的汇编代码，这种汇编并不对应某种真实的硬件架构。之后 Go 的汇编器使用这种伪汇编，为目标硬件生成具体的机器指令。伪汇编这一个额外层可以带来很多好处，最主要的一点是方便将Go移植到新的架构上。

## Go编译指示

### `//go:noinline`

`Inline`，是在编译期间发生的，将函数调用调用处替换为被调用函数主体的一种编译器优化手段。使用内联有如下的好处：

- 减少函数调用的开销，提高执行速度。
- 复制后的更大函数体为其他编译优化带来可能性，如 [过程间优化](https://link.segmentfault.com/?url=https%3A%2F%2Fen.wikipedia.org%2Fwiki%2FInterprocedural_optimization)
- 消除分支，并改善空间局部性和指令顺序性，同样可以提高性能。

也有如下的缺点：

- 代码复制带来的空间增长。
- 如果有大量重复代码，反而会降低缓存命中率，尤其对 CPU 缓存是致命的。

例如下面的代码：

```go
func appendStr(word string) string {
    return "new " + word
}
```

执行 `GOOS=linux GOARCH=386 go tool compile -S main.go > main.S`
部分展出它编译后的样子：

```go
    0x0015 00021 (main.go:4)    LEAL    ""..autotmp_3+28(SP), AX
    0x0019 00025 (main.go:4)    PCDATA    $2, $0
    0x0019 00025 (main.go:4)    MOVL    AX, (SP)
    0x001c 00028 (main.go:4)    PCDATA    $2, $1
    0x001c 00028 (main.go:4)    LEAL    go.string."new "(SB), AX
    0x0022 00034 (main.go:4)    PCDATA    $2, $0
    0x0022 00034 (main.go:4)    MOVL    AX, 4(SP)
    0x0026 00038 (main.go:4)    MOVL    $4, 8(SP)
    0x002e 00046 (main.go:4)    PCDATA    $2, $1
    0x002e 00046 (main.go:4)    LEAL    go.string."hello"(SB), AX
    0x0034 00052 (main.go:4)    PCDATA    $2, $0
    0x0034 00052 (main.go:4)    MOVL    AX, 12(SP)
    0x0038 00056 (main.go:4)    MOVL    $5, 16(SP)
    0x0040 00064 (main.go:4)    CALL    runtime.concatstring2(SB)
```

可以看到，它并没有调用 `appendStr` 函数，而是直接把这个函数体的功能内联了。

如果你不想被内联，怎么办呢？此时就该使用 `go//:noinline` 了，像下面这样写：

```go
//go:noinline
func appendStr(word string) string {
    return "new " + word
}
```

编译后是：

```go
    0x0015 00021 (main.go:4)    LEAL    go.string."hello"(SB), AX
    0x001b 00027 (main.go:4)    PCDATA    $2, $0
    0x001b 00027 (main.go:4)    MOVL    AX, (SP)
    0x001e 00030 (main.go:4)    MOVL    $5, 4(SP)
    0x0026 00038 (main.go:4)    CALL    "".appendStr(SB)
```

此时编译器就不会做内联，而是直接调用 `appendStr` 函数。

### `//go:nosplit`

`nosplit` 的作用是：**跳过栈溢出检测。** 正是因为一个 Goroutine 的起始栈大小是有限制的，且比较小的，才可以做到支持并发很多 Goroutine，并高效调度。
[stack.go](https://link.segmentfault.com/?url=https%3A%2F%2Fgithub.com%2Fgolang%2Fgo%2Fblob%2Fmaster%2Fsrc%2Fruntime%2Fstack.go%23L71) 源码中可以看到，`_StackMin` 是 2048 字节，也就是 2k，它不是一成不变的，当不够用时，它会动态地增长。那么，必然有一个检测的机制，来保证可以及时地知道栈不够用了，然后再去增长。回到话题，`nosplit` 就是将这个跳过这个机制。

**显然地，不执行栈溢出检查，可以提高性能，但同时也有可能发生 `stack overflow` 而导致编译失败。**

### `//go:noescape`

`noescape` 的作用是：**禁止逃逸，而且它必须指示一个只有声明没有主体的函数。** 

**什么是逃逸？** Go 相比 C、C++ 是内存更为安全的语言，主要一个点就体现在它可以自动地将超出自身生命周期的变量，从函数栈转移到堆中，逃逸就是指这种行为。

**禁止逃逸的优劣** ：最显而易见的好处是，GC 压力变小了。因为它已经告诉编译器，下面的函数无论如何都不会逃逸，那么当函数返回时，其中的资源也会一并都被销毁。**不过，这么做代表会绕过编译器的逃逸检查，一旦进入运行时，就有可能导致严重的错误及后果。**

### `//go:norace`

`norace` 的作用是：**跳过竞态检测**。我们知道，在多线程程序中，难免会出现数据竞争，正常情况下，当编译器检测到有数据竞争，就会给出提示。

**禁止静态检测的优劣** ：使用 `norace` 除了减少编译时间，我想不到有其他的优点了。但缺点却很明显，那就是数据竞争会导致程序的不确定性。

### 总结

**绝大多数情况下，无需在编程时使用 `//go:` Go 语言的编译器指示，除非你确认你的程序的性能瓶颈在编译器上，否则你都应该先去关心其他更可能出现瓶颈的事情。**

## 待完善

