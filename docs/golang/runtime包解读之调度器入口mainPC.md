# runtime包解读——调度器的入口mainPC

在`asm_wasm.s`我们知道，在所有的初始化完成之后，golang world开始运转，此时程序会跳转到入口主函数`mainPC`位置：

```assembly
// asm_wasm.s
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

可以看到初始化完成之后，会通过`MOVD`将`mainPC`压入`SP`栈顶，`mainPC`的声明同样在`asm_wasm.s`中：

```assembly
DATA  runtime·mainPC+0(SB)/8,$runtime·main(SB)
GLOBL runtime·mainPC(SB),RODATA,$8
```

在程序跳转指示中，可以看到`mainPC`实际上就是指向`runtime.main`，即调度器的入口主函数。

```assembly
runtime.mainPC -> runtime.main
```



