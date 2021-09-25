# Red

<!-- vscode-markdown-toc -->
* 1. [字符串(string)](#string)
	* 1.1. [ Redis中的自增命令和自减命令](#Redis)
	* 1.2. [Redis中处理字串和二进制位的命令](#Redis-1)
* 2. [列表(list)](#list)
	* 2.1. [非阻塞式操作](#)
	* 2.2. [阻塞式操作](#-1)
* 3. [集合(Set)](#Set)
* 4. [散列(Hash)](#Hash)
* 5. [有序集合](#-1)
* 6. [排序](#-1)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->

##  1. <a name='string'></a>字符串(string)

Redis中的字符串就是一个个字节组成的序列，它可以存储以下三种类型的值：

1. 字节串([`byte string`]())
2. 整数
3. 浮点数

###  1.1. <a name='Redis'></a> Redis中的自增命令和自减命令

Redis可以将字符串形式的数据解析成数字类型并且执行增加或者减少操作，其自增/自减命令如下：

|             命令             |        说明         |
| :--------------------------: | :-----------------: |
|       `INCR key-name`        |     默认将值加1     |
|       `DECR key-name`        |     默认将值减1     |
|   `INCRBY key-name value`    |     将值加value     |
|    `DECR key-name value`     |    将值减少value    |
| `INCRBYFLOAT key-name value` | 将值加上浮点数value |

###  1.2. <a name='Redis-1'></a>Redis中处理字串和二进制位的命令

|                     命令                     |                                说明                                 |
| :------------------------------------------: | :-----------------------------------------------------------------: |
|           `APPEND key-name value`            |                             值尾部追加                              |
|        `GETRANGE key-name start end`         |                     获取`[start, end]`内的字串                      |
|           `GETBIT key-name offset`           |        将字符串看成是二进制位，获取offset位置处的二进制位值         |
|        `SETBIT key-name offset value`        |         将offset位置处的二进制位设置为value(value为0或者1)          |
|       `BITCOUNT key-name [start end]`        |            统计`[start, end]`范围内的**为1的**二进制位数            |
| `BITOP opt dest-key key-name [key-name ...]` | 对一个或者多个二进制位串执行包括AND, OR, XOR, NOT在内的按位运算操作 |
|       `SETRANGE key-name offset value`       |         从offset位置开始设置为value对应的值(只是位置做替换)         |

##  2. <a name='list'></a>列表(list)

###  2.1. <a name=''></a>非阻塞式操作

|                命令                |                   说明                   |
| :--------------------------------: | :--------------------------------------: |
| `RPUSH key-name value [value ...]` |      将一个或者多个值推入列表的右端      |
| `LPUSH key-name value [value ...]` |      将一个或者多个值推入列表的左端      |
|          `RPOP key-name`           |            移除列表最右端的值            |
|          `LPOP key-name`           |            移除列表最左端的值            |
|     `LINDEX key-name offsert`      |      返回列表中偏移量为offset的元素      |
|    `LRANGE key-name start end`     | 返回列表中`[start, end]`范围内的所有元素 |
|     `LTRIM key-name start end`     |    移除`(start, end)`范围内的所有元素    |

###  2.2. <a name='-1'></a>阻塞式操作

|                  命令                   |                                   说明                                    |
| :-------------------------------------: | :-----------------------------------------------------------------------: |
| `BLPOP key-name [key-name ...] timeout` | 从第一个非空列表中弹出最左端的元素，或者在timeout内阻塞直到可弹出元素出现 |
| `BRPOP key-name [key-name ...] timeout` |                                   同上                                    |
|     `RPOPLPUSH source-key dest-key`     |            从source-key中弹出最右端元素并推入dest-key的最左端             |
|    `BRPOPLPUSH source-key dest-key`     |                             弹出，推入并返回                              |

##  3. <a name='Set'></a>集合(Set)

|                      命令                      |                                      说明                                      |
| :--------------------------------------------: | :----------------------------------------------------------------------------: |
|        `SADD key-name item [item ...]`         |                                向集合中添加元素                                |
| `SDIFFSTORE dest-key key-name [key-name ...]`  |       将哪些存在于第一个集合但不存在于**剩下集合**的元素添加到dest-key中       |
|        `SINTER key-name [key-name ...]`        |                                     求交集                                     |
|        `SDIFF key-name [key-name ...]`         |                                     同上上                                     |
|        `SREM key-name item [item ...]`         |              从集合中移除一个或者多个元素，并返回删除的元素的数量              |
|           `SISMEMBER key-name item`            |                    检查item是否是key-name对应的集合中的元素                    |
|                `SCARD key-name`                |                       返回key-name对应的集合中的元素数量                       |
|              `SMEMBERS key-name`               |                          返回key-name集合中的所有元素                          |
|         `SRANDMEMBER key-name [count]`         |                    从集合中随机返回count个元素，默认返回1个                    |
|                `SPOP key-name`                 |                           随机地移除集合中的一个元素                           |
|        `SMOVE source-key dest-key item`        | 如何集合source-key中包含item，移除item并添加到dest-key中，命令返回1，否则返回0 |
| `SINTERSTORE dest-key key-name [key-name ...]` |                                    字面意思                                    |
|        `SUNION key-name [key-name ...]`        |                                     求并集                                     |
| `SUNIONSTORE dest-key key-name [key-name ...]` |                                  求并集并存储                                  |

##  4. <a name='Hash'></a>散列(Hash)

|                    命令                    |             说明             |
| :----------------------------------------: | :--------------------------: |
|       `HMGET key-name key [key ...]`       | 从散列里面获取一个或者多个键 |
| `HMSET key-name key value [key value ...]` |            设置KV            |
|       `HDEL key-name key [key ...]`        |            删除KV            |
|              `HLEN key-name`               |  返回散列包含的键值对的数量  |
|           `HEXISTS key-name key`           |       检查key是否存在        |
|              `HKEYS key-name`              |      获取散列中所有的K       |
|              `HVALS key-name`              |      获取散列中所有的V       |
|             `HGETALL key-name`             |         获取所有的KV         |
|      `HINCRBY key-name key increment`      |            增加K             |
|   `HINCREBYFLOAT key-name key increment`   |        增加float大小         |

##  5. <a name='-1'></a>有序集合

|                                                  命令                                                  |                          说明                          |
| :----------------------------------------------------------------------------------------------------: | :----------------------------------------------------: |
|                            `ZADD key-name score member [score member ...]`                             |         将带有给定分值的成员添加到有序集合里面         |
|                                  `ZREM key-name member [member ...]`                                   |   从有序集合中移除给定的成员，并返回被移除的成员数量   |
|                                           `ZCARD  key-name`                                            |                返回有序集合中的元素数量                |
|                                  `ZINCRBY key-name increment member`                                   |           将memeber成员的score增加increment            |
|                                       `ZCOUNT key-name min max`                                        |           返回分值介于min和max之间的成员数量           |
|                                        `ZRANK key-name member`                                         |                    返回member的排名                    |
|                                        `ZSCORE key-name member`                                        |                  返回member成员的分值                  |
|                               `ZRANGE key-name start stop [WITHSCORES]`                                |        返回有序集合中介于start和stop排名的成员         |
|                                       `ZREVRANK key-name member`                                       |                      返回降序排名                      |
|                              `ZREVRANGE key-name start stop [WITHSCORES]`                              | 返回有序集合给定排名范围内的成员，按照分值从大到小排列 |
|                    `ZREVRANGEBYSCORE key min max [WITHSCORES] [LIMIT offset count]`                    |   给定[min, max]内的所有成员，并按照排名从大到小排列   |
|                                 `ZREMRANGEBYSCORE key-name start stop`                                 |         删除[start, stop]分值范围内的所有成员          |
|                                 `ZREMRANGEBYRANK key-name start stop`                                  |           删除[start, stop]排名内的所有成员            |
|  `ZINTERSTORE dest-key key-count key [key ...] [WEIGHTS weight [weight ...]] [AGGREGATE SUM|MIN|MAX]`  |         对给定的有序集合做类似于集合的交集运算         |
| `ZUNIONSTORE dest-key   key-count key [key ...] [WEIGHTS weight [weight ...]] [AGGREGATE SUM|MIN|MAX]` |         对给定的有序集合做类似于集合的并集运算         |

##  6. <a name='-1'></a>排序

```markdown
SORT source-key [BY pattern] [LIMIT offset count] [GET pattern [GET pattern ...] [ASC|DESC] [ALPHA] [STORE dest-key]]
```



