# gRPC之passthrough解析器

## 何时会使用

* 当gRPC客户端在执行Dial的时候，假如没有指定链接地址的schema，将会默认使用passthrough解析器，schema即访问地址前缀字符串，例如`dns://www.baidu.com/xxx.yy` ，schema就是dns。
* 当用户指定使用passthrough解析器