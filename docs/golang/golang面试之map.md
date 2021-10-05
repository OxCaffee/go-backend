# golang面试之map

## 什么类型的值可以作为map的key

从整体上来说，只要是支持比较的类型都可以作为key，除开slice, map, functions这几种类型，其他类型都是可行的、

而**任何类型都可以作为map的value** 。

## math.NaN作为key

下面的程序代码输出几行?

```go
func NanAsKey() {
	m := make(map[interface{}]interface{})
	m[math.NaN()] = 1
	m[math.NaN()] = 2

	for k, v := range m {
		fmt.Println(k, v)
	}
}
```

程序最后输出2行：

```go
PS D:\Github\repo\Go-Backend\src\g_map> go run main.go
NaN 2
NaN 1
```

**因为NaN有如下特性** ：

* `NaN != NaN`

* `hash(NaN) != hash(NaN)`

## map的key为什么是无序的

因为map会发生扩容，原先的一些bucket中的元素可能搬迁到另外一个完全不同的位置(也取决于被hash的值)，因此是无序的。

## map是线程安全的吗

map不是线程安全的，在查找，赋值，遍历，删除的过程中都会检测写标志，一旦发现写标志置位为1，则直接panic

