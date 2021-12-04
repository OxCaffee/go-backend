# runtime包解读之Go程序启动流程

<!-- vscode-markdown-toc -->
* 1. [前言](#)
* 2. [Golang程序的入口——rt0_xxxx_amd64.s](#Golangrt0_xxxx_amd64.s)
* 3. [Go引导的流程图](#Go)
* 4. [总结](#-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name=''></a>前言

这篇文章对于后续理解Go的调度器模型有很好的意义，因为在调度器启动的过程中，很多函数都是调用的汇编代码，单单看源码的话没有办法将整个流程给串联起来，因此，先了解整个Golang程序的启动流程是很重要的。

##  2. <a name='Golangrt0_xxxx_amd64.s'></a>Golang程序的入口——rt0_xxxx_amd64.s

对于不同计算机架构，例如MacOS或者Windows，Go的程序入口时不同的，例如：

* MacOS在`src/runtime/rt0_darwin_amd64.s`
* Linux在`src/runtime/rt0_linux_amd64.s`
* Windows在`src/runtime/rt0_windows_amd64.s`

`rt0`是`runtime0`的缩写，代表**运行时的创世，上帝** 。

以linux操作系统为例：

```assembly
TEXT _rt0_amd64_linux(SB),NOSPLIT,$-8
	JMP	_rt0_amd64(SB)
```

会发现`rt0_linux_amd64`最终跳转到了`_rt0_amd64` ，其实无论对于哪种架构，最终入口都会被指向为`_rt0_amd64`。`_rt0_amd64`在`runtime/asm_amd64.s`文件中，内容如下：

```assembly
TEXT _rt0_amd64(SB),NOSPLIT,$-8
	MOVQ	0(SP), DI	// argc
	LEAQ	8(SP), SI	// argv
	JMP	runtime·rt0_go(SB)
```

该代码的作用是将输入的`argc`和`argv`从内存移入到寄存器当中，执行完成之后，栈指针SP的前两个值分别为`argc`和`argv`，其对应参数的数量和具体参数值。

`_rt0_amd64`设置完`argc`和`argv`之后将程序跳转到`rt0_go`里。`rt0_go`位于`runtime/asm_wasm.s`中：

```assembly
TEXT runtime·rt0_go(SB), NOSPLIT|NOFRAME|TOPFRAME, $0
	// save m->g0 = g0
	MOVD $runtime·g0(SB), runtime·m0+m_g0(SB)
	// save m0 to g0->m
	MOVD $runtime·m0(SB), runtime·g0+g_m(SB)
	// set g to g0
	MOVD $runtime·g0(SB), g
	CALLNORESUME runtime·check(SB)
	CALLNORESUME runtime·args(SB)
	CALLNORESUME runtime·osinit(SB)
	CALLNORESUME runtime·schedinit(SB)
	MOVD $runtime·mainPC(SB), 0(SP)
	CALLNORESUME runtime·newproc(SB)
	CALL runtime·mstart(SB) // WebAssembly stack will unwind when switching to another goroutine
	UNDEF
```

- `runtime.check`：运行时类型检查，主要是校验编译器的翻译工作是否正确，是否有 “坑”。基本代码均为检查 `int8` 在 `unsafe.Sizeof` 方法下是否等于 1 这类动作。
- `runtime.args`：系统参数传递，主要是将系统参数转换传递给程序使用。
- `runtime.osinit`：系统基本参数设置，主要是获取 CPU 核心数和内存物理页大小。
- `runtime.schedinit`：进行各种运行时组件的初始化，包含调度器、内存分配器、堆、栈、GC 等一大堆初始化工作。会进行 p 的初始化，并将 m0 和某一个 p 进行绑定。
- `runtime.main`：主要工作是运行 main goroutine，虽然在`runtime·rt0_go` 中指向的是`$runtime·mainPC`，但实质指向的是 `runtime.main`。
- `runtime.newproc`：创建一个新的 goroutine，且绑定 `runtime.main` 方法（也就是应用程序中的入口 main 方法）。并将其放入 m0 绑定的p的本地队列中去，以便后续调度。
- `runtime.mstart`：启动 m，调度器开始进行循环调度。

在 `runtime·rt0_go` 方法中，其主要是完成各类运行时的检查，系统参数设置和获取，并进行大量的 Go 基础组件初始化。

初始化完毕后进行主协程（main goroutine）的运行，并放入等待队列（GMP 模型），最后调度器开始进行循环调度。

##  3. <a name='Go'></a>Go引导的流程图

<div align=center><img src="/assets/sched2.png"/></div>

##  4. <a name='-1'></a>总结

以上就是golang程序的引导和启动过程，有了上面大概的了解，我们对于后面的`schedinit`，`mstart`等就有了更深的理解，后面的`runtime`包源码阅读也会按照上面启动的顺序进行。