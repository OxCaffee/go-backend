# Go Modules官方文档解读

## 介绍

模块(Modules)是Go语言管理依赖的方式。

这个文档详细地阐述了Go语言的模块。

* 如果你想了解如何创建一个Go项目(Project)，请移步[如何写Go代码]()
* 如果你想使用Go模块以及重构系统为Go模块，请关注一系列以[使用Go模块]()为开头的博客

## 模块，包和版本

**一个模块是_发布_,_版本控制_和_分发_的包的集合** 。模块可能从版本控制仓库和其他代理服务器中直接下载。

一个模块由`module path`定义，`module path`被声明在`go.mod`文件中，伴随着的还有其他的一些信息，例如该模块的依赖。模块的根文件夹`root directory`是含有`go.mod`文件的文件夹。主模块`Main Module`是go命令调用的文件夹。

一个模块中的每一个包`package`都是一批源文件的集合，它们位于同一个文件夹中，被一起编译。一个包的路径是相较于包含它的包的路径的相对路径。

## 模块路径

模块路径应该描述模块做什么以及在哪里找到它。通常，模块路径由存储库根路径、存储库中的目录(通常为空)和主版本后缀(仅适用于主版本2或更高版本)组成。

* 存储库根路径是模块路径的一部分，它对应于开发模块的版本控制存储库的根目录。大多数模块都是在它们的存储库的根目录中定义的，所以这通常是整个路径。例如，golang.org/x/net是同名模块的存储库根路径。有关go命令如何使用从模块路径派生的HTTP请求来定位存储库的信息，请参见[查找存储库中的模块路径](https://golang.org/ref/mod#vcs-find)。

* 如果模块没有在存储库的根目录中定义，模块子目录是模块路径中命名该目录的部分，不包括主版本后缀。这也可以作为语义版本标记的前缀。例如，模块golang.org/x/tools/gopls位于存储库的gopls子目录中，根路径为golang.org/x/tools，因此它有模块子目录gopls。

* 如果模块在主版本2或更高版本发布，则模块路径必须以主版本后缀/v2结尾。这可能是子目录名的一部分，也可能不是。例如，路径为golang.org/x/repo/sub/v2的模块可能位于存储库golang.org/x/repo的/sub或/sub/v2子目录中。

如果一个模块可能被其他模块所依赖，那么必须遵循这些规则，以便go命令可以找到并下载该模块。

在模块路径中允许的字符也有一些[词法限制](https://golang.org/ref/mod#go-mod-file-ident)。

## 版本

版本标识模块的不可变快照，可以是发布版或预发布版。每个版本都以字母v开头，后面是语义版本。

总而言之，**语义版本由三个非负整数组成(从左到右依次是主要、次要和补丁版本)，它们之间用点隔开** 。补丁版本之后可能会有一个以连字符开头的可选预发布字符串。预发布字符串或补丁版本之后可能会有一个以加号开头的构建元数据字符串。例如，v0.0.0、v1.12.134、v8.0.5-pre和v2.0.9+meta是有效的版本。

版本的每个部分都表明该版本是否稳定，是否与以前的版本兼容。

* 在对模块的公共接口或文档化功能做了向后不兼容的更改之后(例如，在删除了一个包之后)，主版本必须增加，次要版本和补丁版本必须设置为零。
* 在进行向后兼容的更改(例如，在添加了一个新函数之后)后，必须增加次要版本，并将补丁版本设置为零。
* 补丁版本必须在不影响模块公共接口的更改之后增加，比如bug修复或优化。
* 预发布后缀表示版本是预发布版本。预发布版本排序在相应的发布版本之前。例如，v1.2.3-pre先于v1.2.3。

如果一个版本的主版本是0，或者它有一个预发布后缀，那么它就被认为是不稳定的。不稳定版本不受兼容性要求的限制。例如，v0.2.0可能与v0.1.0不兼容，而v1.5.0-beta可能与v1.5.0不兼容。

Go可以使用不遵循这些约定的标记、分支或修订来访问版本控制系统中的模块。然而，在主模块中，go命令将自动将不遵循此标准的修订名转换为规范版本。作为此过程的一部分，go命令还将删除构建元数据后缀(+不兼容的除外)。这可能会产生一个伪版本、一个预发布版本，它编码修订标识符(例如Git提交散列)和一个来自版本控制系统的时间戳。例如，命令`go get -d golang.org/x/net@daa7c041`将提交哈希`daa7c041`转换为伪版本`v0.0.0-20191109021931-daa7c04131f5`。在主模块之外需要规范版本，如果在go中出现master等非规范版本，则go命令将在`go.mod`中报告一个错误。

## 伪版本

伪版本是一种特殊格式化的预发布版本，它对版本控制存储库中关于特定修订的信息进行编码。例如`v0.0.0-20191109021931-daa7c04131f5`是伪版本。

伪版本可能指的是没有语义版本标签可用的修订。它们可以用于在创建版本标记之前测试提交，例如，在一个开发分支上。

每一个伪版本存在下面三个部分：

* 基本版本前缀(`vX.0.0`或`vX.Y.Z-0`)，它来源于修订之前的语义版本标记，或者(如果没有这样的标记)`vX.0.0`。
* 一个时间戳(yyyymmddhhmmss)，它是创建修订的UTC时间。在Git中，这是提交时间，而不是作者时间。
* 修订标识符(abcdefabcdef)，它是提交散列的12个字符前缀，或者在Subversion中，是一个填充了零的修订号。

每个伪版本可能有三种形式之一，这取决于基本版本。这些表单确保伪版本比它的基础版本高，但比下一个标记版本低。

* `vX.0.0-yyyymmddhhmmss-abcdefabcdef`在没有已知基本版本时使用。与所有版本一样，主版本X必须匹配模块的主版本后缀。

* `vX.Y.Z-pre.0`。`yyyymmddhhmmss-abcdefabcdef`在基本版本是预发布版本(如`vX.Y.Z-pre`)时使用。

* `vX.Y。(Z + 1) 0`。``yyyymmddhhmmss-abcdefabcdef`在基本版本是发布版本(如`vX.Y.Z`)时使用。例如，如果基本版本是`v1.2.3`，那么伪版本可能是`v1.2.4-0.20191109021931-daa7c04131f5`。

## 将包解析为模块

当go命令使用包路径加载包时，它需要确定哪个模块提供了包。

go命令首先在编译列表中搜索带有包路径前缀的模块。例如，如果包`example.com/a/b`是导入的，并且`example.com/a`在编译列表中，go命令会检查`example.com/a/`是否包含包b。**至少一个以`*.go*为后缀的目录可以被认为是一个包`** 。编译约束不应用于此目的。**如果编译列表中恰好有一个模块提供了包，则使用该模块。如果没有模块提供该包，或者两个或者两个以上的模块提供该包，那么go将会报告一个错误**。`-mod=mod`标志指示go命令试图找到缺少包的新模块并更新`go.mod`和`go.sum` 。`go get`命令和`go mod tidy`命令会自动完成这些功能。

当go命令在一个新模块中查找路径的时候吗，它会检查`GOPROXY`环境变量，该变量是一个逗号分隔的代理url列表，或者是`direct`或`off`关键字。代理URL表示go命令应该使用`GOPROXY`协议联系模块代理。`direct`表示应该和版本控制系统通信。`off`表示不会尝试通信。`GOPRIVATE`和`GONOPROXY`环境变量也可以代替`off` 。

对于`GOPROXY`中的列表中的每个条目，go命令请求可能提供包的每个模块路径的最新版本(即包的每个前缀)。对于每个成功请求的模块路径，go命令将会**下载最新版本的模块，并检查该模块是否包含所请求的包** ：

* 如果找到一个或者多个模块包含请求的包，使用路径最长的那个
* 如果找到一个或者多个模块的包，但是没有一个包含所请求的包，那么将报告一个错误
* 如果没有找到该模块，go命令将**尝试GOPROXY列表的下一个条目**
* 如果没有一个条目满足，报告错误

例如，假设go命令寻找包`golang.org/x/net/html` 并且`GOPROXY`设置为`https://corp.example.com,https://proxy.golang.org` 。那么go命令会发出如下请求：

* 给`https://corp.example.com` 并行发送：
  * 对`golang.org/x/net/html`的最新版本请求
  * 对`golang.org/x/net`的最新版本请求
  * 对`golang.org/x`的最新版本请求
  * 对`golang.org`的最新版本请求
* 给`https://proxy.golang.org` ，如果前面的请求全部是404或者410错误：
  * 对`golang.org/x/net/html`的最新版本请求
  * 对`golang.org/x/net`的最新版本请求
  * 对`golang.org/x`的最新版本请求
  * 对`golang.org`的最新版本请求

在找到合适的模块之后，go命令会将新模块的路径和版本添加到主模块的`go.mod`中。这确保了将来加载相同包的时候，相同的模块将在相同的版本中使用。如果被解析的包没有被主模块中的包导入，那么新的需求会有一个`//indirect`注释。

## `go.mod`文件

一个模块由一个被UTF8编码的目录根路径下的`go.mod`文件定义，例如：

```go
module example.com/my/thing

go 1.12

require example.com/other/thing v1.0.2
require example.com/new/thing/v2 v2.3.4
exclude example.com/old/thing v1.2.3
replace example.com/bad/thing v1.4.5 => example.com/good/thing v1.4.5
retract [v1.9.0, v1.9.5]
```

还可以简写为：

```go
require (
    example.com/new/thing/v2 v2.3.4
    example.com/old/thing v1.2.3
)
```

`go.mod`文件被设计为可阅读和机器可写的形式。go命令提供了几个子命令来修改`go.mod`文件。例如，`go get`能升级或者降级特定的依赖版本。加载模块图的命令会自动更新`go.mod` 。`go mod edit`可以执行低级修改。`golang.org/x/mod/modfile`还提供了一些程序性修改mod文件的方式。

主模块和任何用本地文件路径指定的替换模块都需要`go.mod`文件。然而，**显式地缺少`go.mod`的模块，仍然可能被作为一个依赖项** 。

## 编译命令

### go build

用法：

```shell
go get [-d] [-t] [-u] [build flags] [packages]
```

例子：

```shell
# Upgrade a specific module.
$ go get -d golang.org/x/net

# Upgrade modules that provide packages imported by packages in the main module.
$ go get -d -u ./...

# Upgrade or downgrade to a specific version of a module.
$ go get -d golang.org/x/text@v0.3.2

# Update to the commit on the module's master branch.
$ go get -d golang.org/x/text@master

# Remove a dependency on a module and downgrade modules that require it
# to versions that don't require it.
$ go get -d golang.org/x/text@none
```

该`go get`命令更新[主模块](https://golang.org/ref/mod#glos-main-module)[`go.mod` 文件](https://golang.org/ref/mod#go-mod-file)中的模块依赖关系，然后构建和安装命令行中列出的包。

第一步是确定要更新哪些模块。`go get`接受包列表、包模式和模块路径作为参数。如果指定了包参数，则`go get`更新提供包的模块。如果指定了包模式（例如，`all`或带有`...` 通配符的路径），则`go get`将该模式扩展为一组包，然后更新提供这些包的模块。如果参数命名模块而不是包（例如，模块`golang.org/x/net`在其根目录中没有包），`go get`将更新模块但不会构建包。如果没有指定参数，`go get`就如同`.`指定了一样（当前目录中的包）；这可以与`-u` 用于更新提供导入包的模块的标志。

每个参数可以包含一个版本查询后缀，指示所需的版本，如`go get golang.org/x/text@v0.3.0`. 版本查询后缀由`@`后跟[版本查询](https://golang.org/ref/mod#version-queries)的符号组成，它可以指示特定版本 ( `v0.3.0`)、版本前缀 ( `v0.3`)、分支或标记名称 ( `master`)、修订版 ( `1234abcd`) 或特殊查询之一`latest`，`upgrade`，`patch`，或`none`。如果没有给出版本，则 `go get`使用`@upgrade`查询。

一旦`go get`将其参数解析为特定模块和版本，`go get`将在主模块的文件中添加、更改或删除[`require`指令](https://golang.org/ref/mod#go-mod-file-require)，`go.mod`以确保模块将来保持所需的版本。请注意，`go.mod`文件中所需的版本是 *最低版本，*并且可能会随着新依赖项的添加而自动增加。有关如何选择版本以及如何通过模块感知命令解决冲突的详细信息，请参阅[最小版本选择 (MVS)](https://golang.org/ref/mod#minimal-version-selection)。

**如果命名模块的新版本需要更高版本的其他模块，则在添加、升级或降级命令行中命名的模块时，可能会升级其他模块。例如，假设模块`example.com/a`升级到版本`v1.5.0`，并且该版本需要模块`example.com/b` 的版本`v1.2.0`。如果`example.com/b`版本当前需要 模块`v1.1.0`，`go get example.com/a@v1.5.0`也将升级`example.com/b`到 `v1.2.0`.**

`go get` 支持以下标志：

- 该`-d`标志告诉`go get`不要构建或安装包。当`-d`使用时，`go get`将只管理依赖关系`go.mod`。`go get` 不`-d`推荐使用without来构建和安装包（从 Go 1.17 开始）。在 Go 1.18 中，`-d`将始终启用。
- 该`-u`标志告诉`go get`升级模块，这些模块提供由命令行上命名的包直接或间接导入的包。选择的每个模块`-u`都将升级到其最新版本，除非更高版本（预发布）已经需要它。
- 该`-u=patch`标志（不`-u patch`）也告诉`go get`升级的依赖，**但`go get`将升级每个依赖于最新的补丁版本（类似`@patch`版本查询）**。
- 该`-t`标志告诉`go get`要考虑构建命令行上命名的包的测试所需的模块。当`-t`和`-u`一起使用时， `go get`也会更新测试依赖项。
- `-insecure`不应再使用该标志。它允许`go get`使用不安全的方案（如 HTTP）解析自定义导入路径并从存储库和模块代理中获取。在`GOINSECURE` [环境变量](https://golang.org/ref/mod#environment-variables)提供了更精细的控制，并应被代替使用。

### go install

用法：

```shell
go install [build flags] [packages]
```

例子：

```shell
# Install the latest version of a program,
# ignoring go.mod in the current directory (if any).
$ go install golang.org/x/tools/gopls@latest

# Install a specific version of a program.
$ go install golang.org/x/tools/gopls@v0.6.4

# Install a program at the version selected by the module in the current directory.
$ go install golang.org/x/tools/gopls

# Install all programs in a directory.
$ go install ./cmd/...
```

该`go install`命令构建并安装由命令行上的路径命名的包。可执行文件（`main`包）安装到由指定的目录`GOBIN`的环境变量，其默认值为 `$GOPATH/bin`或者`$HOME/go/bin`如果`GOPATH`环境变量没有设置。可执行文件在`$GOROOT`安装在`$GOROOT/bin`或`$GOTOOLDIR`代替`$GOBIN`。

**从 Go 1.16 开始，如果参数具有版本后缀（如`@latest`或 `@v1.0.0`），`go install`则以模块感知模式构建包，忽略 `go.mod`当前目录或任何父目录中的文件（如果有）。这对于在不影响主模块的依赖关系的情况下安装可执行文件很有用。**

为了消除在构建中使用哪些模块版本的歧义，参数必须满足以下约束：

- 参数必须是包路径或包模式（带有“ `...`”通配符）。它们不能是标准包（如`fmt`）、元模式（`std`、`cmd`、 `all`），或者相对或绝对文件路径。
- 所有参数必须具有相同的版本后缀。不允许不同的查询，即使它们引用相同的版本。
- 所有参数必须引用同一模块中同一版本中的包。
- 没有模块被认为是[主模块](https://golang.org/ref/mod#glos-main-module)。如果包含在命令行上命名的包的模块有一个`go.mod`文件，它不能包含指令（`replace`和`exclude`），这会导致它与主模块的解释不同。该模块不得要求其自身的更高版本。
- 包路径参数必须引用`main`包。模式参数只会匹配`main`包。

Go 1.15 及更低版本不支持使用`go install`.

如果参数没有版本后缀，则`go install`可能以模块感知模式或`GOPATH`模式运行，具体取决于`GO111MODULE`环境变量和`go.mod`文件的存在。有关详细信息，请参阅[模块感知命令](https://golang.org/ref/mod#mod-commands)。如果启用了模块感知模式，`go install`则在主模块的上下文中运行，这可能与包含正在安装的包的模块不同。

### go list -m

用法：

```shell
go list -m [-u] [-retracted] [-versions] [list flags] [modules]
```

例子：

```shell
$ go list -m all
$ go list -m -versions example.com/m
$ go list -m -json example.com/m@latest
```

该`-m`标志导致`go list`列出模块而不是包。在这种模式下， 的参数`go list`可能是模块、模块模式（包含 `...`通配符）、[版本查询](https://golang.org/ref/mod#version-queries)或特殊模式 `all`，它匹配[构建列表](https://golang.org/ref/mod#glos-build-list)中的所有模块。如果未指定参数，则列出[主模块](https://golang.org/ref/mod#glos-main-module)。

在列出模块时，该`-f`标志仍然指定应用于 Go 结构体的格式模板，但现在是一个`Module`结构体：

```go
type Module struct {
    Path       string       // module path
    Version    string       // module version
    Versions   []string     // available module versions (with -versions)
    Replace    *Module      // replaced by this module
    Time       *time.Time   // time version was created
    Update     *Module      // available update, if any (with -u)
    Main       bool         // is this the main module?
    Indirect   bool         // is this module only an indirect dependency of main module?
    Dir        string       // directory holding files for this module, if any
    GoMod      string       // path to go.mod file for this module, if any
    GoVersion  string       // go version used in module
    Deprecated string       // deprecation message, if any (with -u)
    Error      *ModuleError // error loading module
}

type ModuleError struct {
    Err string // the error itself
}
```

默认输出是打印模块路径，然后打印有关版本和替换的信息（如果有）。例如，`go list -m all`可能会打印：

```shell
example.com/main/module
golang.org/x/net v0.1.0
golang.org/x/text v0.3.0 => /tmp/text
rsc.io/pdf v0.1.1
```

该`Module`结构有一个`String`进行格式化这条线的输出，以便默认格式等同于方法`-f '{{.String}}'`。

请注意，当模块被替换时，其`Replace`字段描述替换模块模块，并且其`Dir`字段设置为替换模块的源代码（如果存在）。（也就是说，如果`Replace`非零，则`Dir` 设置为`Replace.Dir`，无法访问被替换的源代码。）

该`-u`标志添加了有关可用升级的信息。当给定模块的最新版本比当前版本新时，`list -u`将模块的 `Update`字段设置为有关较新模块的信息。`list -u`还打印当前选择的版本 是否已[撤回](https://golang.org/ref/mod#glos-retracted-version)以及模块是否[已弃用](https://golang.org/ref/mod#go-mod-file-module-deprecation)。该模块的`String`方法通过在当前版本之后的括号中格式化较新版本来指示可用升级。例如，`go list -m -u all` 可能会打印：

```shell
example.com/main/module
golang.org/x/old v1.9.9 (deprecated)
golang.org/x/net v0.1.0 (retracted) [v0.2.0]
golang.org/x/text v0.3.0 [v0.4.0] => /tmp/text
rsc.io/pdf v0.1.1 [v0.1.2]
```

（对于工具，`go list -m -u -json all`可能更方便解析。）

该`-versions`标志导致`list`将模块的`Versions`字段设置为该模块的所有已知版本的列表，根据语义版本控制从低到高排序。该标志还会更改默认输出格式以显示模块路径，后跟以空格分隔的版本列表。除非`-retracted`还指定了标志，否则此列表中将省略撤回的版本。

该`-retracted`标志指示`list`在与该`-versions`标志一起打印的列表中显示收回的版本，并在解决[版本查询](https://golang.org/ref/mod#version-queries)时考虑收回的版本。例如，`go list -m -retracted example.com/m@latest`显示模块的最高发行版或预发行版`example.com/m`，即使该版本已撤回。 在此版本中，从文件中加载[`retract`指令](https://golang.org/ref/mod#go-mod-file-retract)和 [弃用](https://golang.org/ref/mod#go-mod-file-module-deprecation)`go.mod`。该`-retracted`标志是在 Go 1.16 中添加的。

模板函数`module`接受一个字符串参数，该参数必须是模块路径或查询，并将指定的模块作为`Module`结构返回。如果发生错误，结果将是一个`Module`带有非 nil`Error` 字段的结构体。

### go mod download

用法：

```shell
go mod download [-json] [-x] [modules]
```

例子：

```shell
$ go mod download
$ go mod download golang.org/x/mod@v0.2.0
```

该`go mod download`命令将命名的模块下载到[模块缓存中](https://golang.org/ref/mod#glos-module-cache)。参数可以是模块路径或模块模式，选择主模块的依赖项或表单的[版本查询](https://golang.org/ref/mod#version-queries)`path@version`。不带参数， `download`适用于[主模块的](https://golang.org/ref/mod#glos-main-module)所有依赖项。

该`go`命令将在正常执行期间根据需要自动下载模块。该`go mod download`命令主要用于预填充模块缓存或加载要由[模块代理](https://golang.org/ref/mod#glos-module-proxy)服务的数据。

默认情况下，`download`不向标准输出写入任何内容。它将进度消息和错误打印到标准错误。

该`-json`标志导致`download`将一系列 JSON 对象打印到标准输出，描述每个下载的模块（或失败），对应于这个 Go 结构：

```go
type Module struct {
    Path     string // module path
    Version  string // module version
    Error    string // error loading module
    Info     string // absolute path to cached .info file
    GoMod    string // absolute path to cached .mod file
    Zip      string // absolute path to cached .zip file
    Dir      string // absolute path to cached source root directory
    Sum      string // checksum for path, version (as in go.sum)
    GoModSum string // checksum for go.mod (as in go.sum)
}
```

该`-x`标志导致`download`打印命令`download`执行到标准错误。

### go mod edit

用法：

```shell
go mod edit [editing flags] [-fmt|-print|-json] [go.mod]
```

例子：

```shell
# Add a replace directive.
$ go mod edit -replace example.com/a@v1.0.0=./a

# Remove a replace directive.
$ go mod edit -dropreplace example.com/a@v1.0.0

# Set the go version, add a requirement, and print the file
# instead of writing it to disk.
$ go mod edit -go=1.14 -require=example.com/m@v1.0.0 -print

# Format the go.mod file.
$ go mod edit -fmt

# Format and print a different .mod file.
$ go mod edit -print tools.mod

# Print a JSON representation of the go.mod file.
$ go mod edit -json
```

该`go mod edit`命令提供了一个用于编辑和格式化`go.mod`文件的命令行界面，主要供工具和脚本使用。`go mod edit` 只读取一个`go.mod`文件；它不查找有关其他模块的信息。默认情况下，`go mod edit`读取和写入`go.mod`主模块的文件，但可以在编辑标志后指定不同的目标文件。

编辑标志指定编辑操作的序列。

- 该`-module`标志更改模块的路径（`go.mod`文件的模块行）。
- 该`-go=version`标志设置了预期的 Go 语言版本。
- 在`-require=path@version`和`-droprequire=path`标志添加和删除指定的模块路径和版本上的要求。请注意，`-require` 会覆盖 上的任何现有要求`path`。这些标志主要用于理解模块图的工具。用户应该更喜欢`go get path@version`或`go get path@none`，`go.mod`根据需要进行其他调整以满足其他模块强加的约束。见[`go get`](https://golang.org/ref/mod#go-get)。
- 在`-exclude=path@version`和`-dropexclude=path@version`标志增加和放弃对给定的模块路径和版本的排除。请注意，`-exclude=path@version`如果该排除已存在， 则为空操作。
- 该`-replace=old[@v]=new[@v]`标志添加了给定模块路径和版本对的替换。如果省略`@v`in `old@v`，则添加左侧没有版本的替换，这适用于旧模块路径的所有版本。如果省略`@v`in `new@v`，则新路径应该是本地模块根目录，而不是模块路径。请注意，`-replace` 覆盖 的任何冗余替换`old[@v]`，因此省略`@v`将删除特定版本的替换。
- 该`-dropreplace=old[@v]`标志丢弃给定模块路径和版本对的替换。如果`@v`提供了，则删除给定版本的替换。左侧没有版本的现有替代品仍可更换模块。如果`@v`省略 ，则删除没有版本的替换。
- 的`-retract=version`和`-dropretract=version`标志添加和删除的缩回对于给定的版本，其可以是一个单一的版本（如 `v1.2.3`）或间隔（等`[v1.1.0,v1.2.0]`）。请注意，该`-retract` 标志不能为`retract`指令添加基本原理注释。推荐理由注释，并且可以通过`go list -m -u`和其他命令显示。

编辑标志可以重复。更改按给定的顺序应用。

`go mod edit` 有额外的标志来控制它的输出。

- 该`-fmt`标志重新格式化`go.mod`文件而不进行其他更改。使用或重写`go.mod`文件的任何其他修改也暗示了这种重新格式化。唯一需要此标志的情况是没有指定其他标志，如`go mod edit -fmt`.
- 该`-print`标志`go.mod`以其文本格式打印 final ，而不是将其写回磁盘。
- 该`-json`标志`go.mod`以 JSON 格式打印最终结果，而不是以文本格式将其写回磁盘。JSON 输出对应于这些 Go 类型：

```go
type Module struct {
        Path    string
        Version string
}

type GoMod struct {
        Module  Module
        Go      string
        Require []Require
        Exclude []Module
        Replace []Replace
}

type Require struct {
        Path     string
        Version  string
        Indirect bool
}

type Replace struct {
        Old Module
        New Module
}

type Retract struct {
        Low       string
        High      string
        Rationale string
}
```

请注意，这仅描述了`go.mod`文件本身，而不是间接引用的其他模块。对于可用于构建的完整模块集，请使用`go list -m -json all`. 见[`go list -m`](https://golang.org/ref/mod#go-list-m)。

例如，一个工具能够获得`go.mod`通过解析的输出文件作为数据结构`go mod edit -json`，然后可以通过调用进行更改 `go mod edit`与`-require`，`-exclude`，等。

工具还可以使用该包 [`golang.org/x/mod/modfile`](https://pkg.go.dev/golang.org/x/mod/modfile?tab=doc) 来解析、编辑和格式化`go.mod`文件。

### go mod graph

用法：

```shell
go mod graph [-go=version]
```

该`go mod graph`命令以文本形式打印[模块需求图](https://golang.org/ref/mod#glos-module-graph)（应用了替换）。例如：

```shell
example.com/main example.com/a@v1.1.0
example.com/main example.com/b@v1.2.0
example.com/a@v1.1.0 example.com/b@v1.1.1
example.com/a@v1.1.0 example.com/c@v1.3.0
example.com/b@v1.1.0 example.com/c@v1.1.0
example.com/b@v1.2.0 example.com/c@v1.2.0
```

模块图中的每个顶点代表一个模块的特定版本。图中的每条边代表对依赖项的最低版本的要求。

`go mod graph`打印图形的边缘，每行一个。每行有两个空格分隔的字段：模块版本及其依赖项之一。每个模块版本都被标识为一个形式为 的字符串`path@version`。主模块没有`@version`后缀，因为它没有版本。

该`-go`标志导致`go mod graph`由给定围棋版本加载，而不是版本指示由报告模块图形[`go`指令](https://golang.org/ref/mod#go-mod-file-go)中`go.mod`的文件。

### go mod init

用法：

```
go mod init [module-path]
```

例子：

```
go mod init
go mod init example.com/m
```

该`go mod init`命令`go.mod`在当前目录中初始化并写入一个新文件，实际上创建了一个以当前目录为根的新模块。该`go.mod`文件必须不存在。

`init`接受一个可选参数，新模块的[模块路径](https://golang.org/ref/mod#glos-module-path)。有关选择[模块路径](https://golang.org/ref/mod#module-path)的说明，请参阅模块路径。如果省略模块路径参数，`init`将尝试使用`.go`文件中的导入注释、vendoring 工具配置文件和当前目录（如果在 中`GOPATH`）推断模块路径。

如果存在 vendoring 工具的配置文件，`init`将尝试从中导入模块需求。`init`支持以下配置文件。

- `GLOCKFILE`
- `Godeps/Godeps.json`
- `Gopkg.lock` 
- `dependencies.tsv` 
- `glide.lock` 
- `vendor.conf` 
- `vendor.yml` 
- `vendor/manifest` 
- `vendor/vendor.json` 

如果同一仓库中的多个包以不同的版本导入，而仓库只包含一个模块，则导入的`go.mod`只能需要一个版本的模块。您可能希望运行[`go list -m all`](https://golang.org/ref/mod#go-list-m)以检查[构建列表](https://golang.org/ref/mod#glos-build-list)中的所有版本，并[`go mod tidy`](https://golang.org/ref/mod#go-mod-tidy)添加缺少的需求并删除未使用的需求。

### go mod tidy

用法：

```
go mod tidy [-e] [-v] [-go=version] [-compat=version]
```

`go mod tidy`确保`go.mod`文件与模块中的源代码匹配。它添加了构建当前模块的包和依赖项所需的任何缺失的模块要求，并删除了对不提供任何相关包的模块的要求。它还添加任何缺失的条目 `go.sum`并删除不必要的条目。

**尽管在加载包时遇到错误，该`-e`标志（在 Go 1.16 中添加）导致`go mod tidy`尝试继续。**

**该`-v`标志导致`go mod tidy`将有关已删除模块的信息打印到标准错误。**

`go mod tidy`通过递归加载[主模块中的](https://golang.org/ref/mod#glos-main-module)所有包和它们导入的所有包来工作。这包括测试导入的包（包括其他模块中的测试）。`go mod tidy`就像启用了所有构建标记一样，因此它会考虑特定于平台的源文件和需要自定义构建标记的文件，即使这些源文件通常不会被构建。有一个例外：`ignore`未启用构建标记，因此`// +build ignore`不会考虑具有构建约束的文件。请注意，`go mod tidy` 不会考虑在包命名的目录在主模块中`testdata`或者与开头的名称`.`或`_`除非这些包明确被其它软件包进口。

一旦`go mod tidy`加载了这组包，它确保提供一个或多个包的每个模块`require`在主模块的`go.mod`文件中都有一个指令，或者，如果主模块位于`go 1.16`或低于（参见 [模块图冗余](https://golang.org/ref/mod#graph-redundancy)），另一个需要的模块需要. `go mod tidy`将在每个缺少的模块上添加对最新版本的要求（请参阅[版本查询](https://golang.org/ref/mod#version-queries)以了解[版本](https://golang.org/ref/mod#version-queries)的定义`latest`）。`go mod tidy`将删除`require`不提供上述集合中任何包的模块的指令。

`go mod tidy`还可以添加或删除`// indirect`对`require` 指令的注释。一个`// indirect`注释表示一个模块，其不提供通过封装在主模块中导入的一个包。（有关何时 添加依赖项和注释的更多详细信息，请参阅[`require` 指令](https://golang.org/ref/mod#go-mod-file-require)`// indirect`。）

如果`-go`设置了标志，`go mod tidy`将根据该版本将[`go` 指令](https://golang.org/ref/mod#go-mod-file-go)更新为[指示](https://golang.org/ref/mod#go-mod-file-go)的版本，启用或禁用 [延迟加载](https://golang.org/ref/mod#lazy-loading)和[模块图修剪](https://golang.org/ref/mod#graph-pruning)（并根据需要添加或删除间接要求）。

默认情况下，当模块图由指令中指示的版本之前的 Go 版本加载时，`go mod tidy`将检查[所选版本](https://golang.org/ref/mod#glos-selected-version)的模块是否不会更改 `go`。也可以通过`-compat`标志明确指定版本检查的兼容性。

### go mod verify

用法：

```shell
go mod verify
```

`go mod verify`检查 存储在[模块缓存](https://golang.org/ref/mod#glos-module-cache)中的[主模块的](https://golang.org/ref/mod#glos-main-module)依赖项自下载以来没有被修改。要执行此检查，请对每个下载的模块[文件](https://golang.org/ref/mod#zip-files)和提取的目录进行散列，然后将这些散列与首次下载模块时记录的散列进行比较。检查[构建列表](https://golang.org/ref/mod#glos-build-list)中的每个模块（可以用 打印）。`go mod verify`[`.zip`](https://golang.org/ref/mod#zip-files)`go mod verify`[`go list -m all`](https://golang.org/ref/mod#go-list-m)

如果所有模块都未修改，则`go mod verify`打印“所有模块已验证”。否则，它将报告哪些模块已更改并以非零状态退出。

请注意，所有模块感知命令都会验证主模块 `go.sum`文件中的散列是否与为下载到模块缓存中的模块记录的散列匹配。如果缺少散列`go.sum`（例如，因为模块是第一次使用），该`go`命令将使用[校验和数据库](https://golang.org/ref/mod#checksum-database)验证其散列 （除非模块路径与`GOPRIVATE`或匹配 `GONOSUMDB`）。有关详细信息，请参阅[验证模块](https://golang.org/ref/mod#authenticating)。

相比之下，`go mod verify`检查模块`.zip`文件及其提取的目录是否具有与首次下载时模块缓存中记录的哈希匹配的哈希。这对于在下载并验证模块*后*检测模块缓存中文件的更改非常有用。`go mod verify` 不下载不在缓存中的`go.sum`模块的内容，也不使用 文件来验证模块内容。但是，`go mod verify`可能会下载 `go.mod`文件以执行[最小版本选择](https://golang.org/ref/mod#minimal-version-selection)。它将`go.sum`用于验证这些文件，并且可能会`go.sum`为丢失的散列添加条目。

### go clean -modcache

用法：

```
go clean [-modcache]
```

该`-modcache`标志导致[`go clean`](https://golang.org/cmd/go/#hdr-Remove_object_files_and_cached_files)删除整个 [模块缓存](https://golang.org/ref/mod#glos-module-cache)，包括版本化依赖项的解压缩源代码。

这通常是移除模块缓存的最佳方式。默认情况下，模块缓存中的大多数文件和目录都是只读的，以防止测试和编辑器在经过[身份验证](https://golang.org/ref/mod#authenticating)后无意中更改文件 。不幸的是，这会导致命令 `rm -r`失败，因为如果不首先使它们的父目录可写，就无法删除文件。

该`-modcacherw`标志（被[`go build`](https://golang.org/cmd/go/#hdr-Compile_packages_and_dependencies)其他模块感知命令接受）导致模块缓存中的新目录可写。要传递`-modcacherw`给所有模块感知命令，请将其添加到 `GOFLAGS`变量中。`GOFLAGS`可以在环境中设置或使用[`go env -w`](https://golang.org/cmd/go/#hdr-Print_Go_environment_information). 例如，下面的命令永久设置它：

```
go env -w GOFLAGS=-modcacherw
```

`-modcacherw`应谨慎使用；开发人员应注意不要更改模块缓存中的文件。[`go mod verify`](https://golang.org/ref/mod#go-mod-verify) 可用于检查缓存中的`go.sum`文件是否与主模块文件中的哈希匹配 。