# MySQL锁总结

## 前言

加锁情况样例表格:

|c1(pk)|c2(uk)|c3(index)|c4|
|:---:|:---:|:---:|:---:|
|10|11|12|13|
|20|21|22|23|
|30|31|32|33|
|40|41|42|43|

其中，c1为主键，c2为唯一索引，c3为普通索引，c4为普通列。

## RC(Read-Commited)和RU(Read-UnCommited)隔离级别

* 查询条件为主键等值查询:

|SQL模版|SQL示例|加锁情况|说明|
|:---:|:---:|:---:|:---:|
|`select ... where pk = ... for update`|`select * from t where c1 = 20 for update`|在c1=20的主键记录上加X锁，即`LOCK_X|LOCK_REC_NOT_GAP`|主键等值查询记录存在|
|`select ... where pk = ... for update`|`select * from t where c1 = 15 for update`|不加锁|在RC和RU级别下，主键等值查询不匹配，不进行加锁|
|`select ... where pk = ... lock in share mode`|`select * from t where c1 = 20 lock in share mode`|在c1=20的主键记录上加S锁，即`LOCK_S|LOCK_REC_NOT_GAP`|主键等值查询记录存在|
|`update ... where pk = ...`|`update t set c4 = 12 where c1 = 20`|在c1=20主键记录上加X锁，即`LOCK_X|LOCK_REC_NOT_GAP`|主键等值查询并更新，记录存在，且未更新索引列|
|`update ... where pk = ...`|`update t set c2 = 12 where c1 = 20`|在c1=20主键记录上加X锁，即`LOCK_X|LOCK_REC_NOT_GAP`，还需要在c2索引上加同样的锁（同理c3)|主键等值查询并更新，记录存在，且更新索引列|
|`delete from ...`|`delete from t where c1 = 20`|对主键，各个索引记录都加上X锁，即`LOCK_X|LOCK_REC_NOT_GAP`|删除表|

* 查询条件为主键范围查询:

**查询条件为范围查询时，需要对匹配的行依次加上等值查询时需要加的锁**

|SQL模版|SQL示例|加锁情况|说明|
|:---:|:---:|:---:|:---:|
|`select ... where pk >= ... for update`|`select * from t where c1 >= 10 for update`|会对c1=`<10, 20, 30, 40>`加上X锁，即`LOCK_X|LOCK_REC_NOT_GAP`|主键范围查询，记录存在|
|`select ... where pk <= ... for update`|`select * from t where c1 <= 20 for update`|初始时刻，会对c1=`<10, 20, 30>`加上X锁，即`LOCK_X|LOCK_REC_NOT_GAP`，后续释放c1=30的锁|主键范围查询，记录存在|
|`update ... where pk >= ...`|`update t set c2 = c2 + 1 where c1 >= 10`|对c1=`<10, 20, 30, 40>`主键记录和c2索引列加上X锁，即`LOCK_X|LOCK_REC_NOT_GAP`|主键范围查询并更新，记录存在|
|`update ... where pk <= ...`|`update t set c2 = c2 + 1 where c1 <= 20`|对c1=`<10, 20, 30>`主键记录和c2索引列加上X锁，即`LOCK_X|LOCK_REC_NOT_GAP`，后续释放c1=30的锁|主键范围查询并更新，记录存在|


* 查询条件为唯一索引等值查询:

|SQL模版|SQL示例|加锁情况|说明|
|:---:|:---:|:---:|:---:|
|`select ... where uk = ... for update`|`select * from t where c2 = 21 for update`|在c2=21唯一索引记录上和主键记录上加X锁，即`LOCK_X|LOCK_REC_NOT_GAP`|唯一索引等值查询，记录存在|
|`select ... where uk = ... for update`|`select * from t where c2 = 20 for update`|不加锁|唯一索引查询条件对应的记录不存在，不会加锁|
|`select ... where uk = ... lock in share mode`|`select * from t where c2 = 21 lock in share mode`|在c2=21唯一索引记录和主键记录上加S锁，即`LOCK_S|LOCK_REC_NOT_GAP`|唯一索引等值查询，记录存在|
|`update ... where uk = ...`|`update t set c4 = 100 where c2 = 21`|对c2=21唯一索引记录和主键记录加上X锁，即`LOCK_X|LOCK_REC_NOT_GAP`，其余索引不加锁|唯一索引等值查询并更新，但不更新索引列|
|`update ... where uk = ...`|`update t set c3 = 100 where c2 = 21`|对c2=21唯一索引记录，c1主键记录，c3索引记录全部加上X锁，即`LOCK_X|LOCK_REC_NOT_GAP`|唯一索引等值查询并更新，更新列包含索引列|
|`delete ... where uk = ...`|`delete from t where c2 = 21`|对c2=21唯一索引记录加X锁，根据唯一索引找到主键，将主键和其余索引全部加上X锁，即`LOCK_X|LOCK_REC_NOT_GAP`|唯一索引等值查询并删除|

* 查询条件为唯一索引范围查询:

|SQL模版|SQL示例|加锁情况|说明|
|:---:|:---:|:---:|:---:|
|`select ... where uk >= ... for update`|`select * from t where c2 >= 21 for update`|不会对唯一索引进行加锁，优化器会判断此时对主键索引加锁性能更高，但初始时刻会对全部主键记录加上X锁，即在c1=`<10, 20, 30, 40>`加上`LOCK_X|LOCK_REC_NOT_GAP` ，后续会释放掉c1=10的锁，因此最终加锁情况是c1=`<20, 30, 40>`|唯一索引范围查询，记录存在|
|`select ... force index(...) where uk >= ... for update`|`select * from t force index(idx_c2) where c2 >= 21 for update`|不会初始时刻全部加锁，然后释放，而是依次对满足条件的记录唯一索引记录和主键记录依次加X锁，即`LOCK_X|LOCK_REC_NOT_GAP`|唯一索引范围查询，记录存在|
|`select ... where uk <= ... for update`|`select * from t where c2 <= 21 for update`|这里`<=`和`>=`情况不一样，此时会对**c2=`<11, 21, 31>`加X锁，并且对c1=`<10, 20>`加X锁，但是不会释放掉c2=31对应的锁** ，原因是索引下推|唯一索引范围查询，记录存在|
|`select ... force index(...) where uk <= ... for update`|`select * from t force index(i_c2) where c2 <= 21 for update`|依然不同于`>=`，会将c2=`<11, 21, 31>`加X锁，对c1=`<10, 20>`主键记录加X锁|唯一索引范围查询，记录存在|
|`update ... where uk >= ...`|`update t set c4 = 100 where c2 >= 21`|对c2=`<21, 31, 41>`和c1=`<20, 30, 40>`全部加X锁，并且不会释放|唯一索引范围查询并更新，记录存在，且不更新索引列|
|`update ... where uk <= ...`|`update t set c4 = 100 where c2 <= 21`|对c2=`<11, 21, 31>`和c1=`<10, 20, 30>`全部加X锁，并且不会释放|唯一索引范围查询并更新，不更新索引列|
|`update ... force index(...) set ... where uk >= ...`|`update t force index(i_c2) set c4 = 100 where c2 >= 21`|对c2=`<21, 31, 41>`和c1=`<20, 30, 40>`主键记录全部加X锁|唯一索引范围查询并更新，且更新字段不属于索引字段|
|`update ... force index(...) set ... where uk <= ...`|`update t force index(i_c2) set c4 = 100 where c2 <= 21`|对c2=`<11, 21, 31>`和c1=`<10, 20, 30>`主键记录全部加X锁，并随机释放c2=31或者c1=30对应的X锁|同上|
|`update ... where uk >= ...`|`update t force index(i_c2) set c3 = 1 where c2 >= 21`|对c1=`<20, 30, 40>`主键记录, c2=`<21, 31, 41>`唯一索引记录，c3=`<22, 32, 42>`索引记录全部加X锁|唯一索引等值范围查询并更新，更新字段包含索引字段|
|`update ... where uk <= ...`|`update t force index(i_c2) set c3 = 1 where c2 <= 21`|对c1=`<10, 20, 30>`主键记录, c2=`<11, 21, 31>`唯一索引记录，c3=`<12, 22>`索引记录全部加X锁，释放c1=30和c2=31的X锁|唯一索引等值范围查询并更新，更新字段包含索引字段|

* 查询条件为非索引列查询:

情况同主键查询一致，因为非索引查询的操作最终都会落到索引上，具体说就是主键索引上。

## RR(Repeatable-Read)隔离级别

RR隔离级别用到了临键锁，即`Next-Key Lock`，其是一个左开右闭的区间，由某个记录的值和该记录与前一个记录之间的间隙组成。例如t表格中的数据，如果对c1=20加临键锁，加锁区间是`(10, 20]`，因为c1=20的前一个记录是c1=10。

数据库进行DML操作的时候，在RR隔离级别下，有如下原则：
* Principle 1: 任何需要访问的数据，都需要加临键锁，也就是加锁的初始粒度是`Next-Key Lock`级别，至于临键锁会不会退化，需要考虑不同的场景。
* Principle 2: `where`条件下，会访问到索引上第一个不满足条件的数据，并且这个数据也会被加上临键锁，至于这个临键锁会不会退化成间隙锁，取决于不同的MySQL版本(最新的MySQL版本会退化)。
* Principle 3: 索引查询，除了**唯一索引等值查询成功**或者**非唯一索引范围查询**这两种情况之外，临键锁都会退化，且第一种情况，临键锁会退化成行锁，第二种情况，临键锁会退化成间隙锁。
* Principle 4: 索引上的等值查询失败，不满足等值条件的后一个元素的临键锁会退化成间隙锁(最新MySQL版本)。

### 查询索引是非唯一索引

|SQL示例|加锁情况|类别|
|:---:|:---:|:---:|
|`select * from t where c3 = 22 for update`|初始时刻会给c3=22加上临键锁，即`LOCK_X|LOCK_ORDINARY`，加锁区间为`(12, 22]`，根据Principle 2，会继续向后查找到第一个不满足条件的值，即c3=32，加上临键锁`LOCK_X|LOCK_ORDINARY`，加锁区间为`(22, 32]`，又根据Principle 3，退化为间隙锁，即`LOCK_X|LOCK_GAP`，加锁区间为`(22, 32)`，因此最后的加锁范围是`(12, 32)`|非唯一索引等值查询成功|
|`select * from t where c3 = 20 for update`|c3=20的记录不存在，根据原则会找到20之后第一个不满足条件的记录，即c3=22，加上间隙锁，即`LOCK_X|LOCK_ORDINARY`，加锁区间为`(12, 22]`，又根据Principle 3，退化成间隙锁，即`LOCK_X|LOCK_GAP`，加锁区间变为`(12, 22)`|非唯一索引等值查询失败|
|`select * from t where c3 > 20 for update`|20是`(12, 22)`之间的一个值，也就是c3中不存在的值，不存在的值没有办法加上临键锁，MySQL会先找到第一个满足这个条件的记录，即c3=22，加上临键锁，即`LOCK_X|LOCK_ORDINARAY`，加锁区间为`(20, 22]`，根据Principle 3，该范围不会退化，又由于没有右区间，因此会加锁`(22, +∞]`，最后的加锁区间为`(12, +∞]`|非唯一索引范围查询，开区间|
|`select * from t where c3 >= 20 for update`|同上，加锁区间为`(12, +∞]`|非唯一索引范围查询，闭区间|
|`select * from t where c3 > 22 and c3 < 24 for update`|对于左区间，加锁范围为`(22, +∞]`，对于右区间，加锁范围为`(-∞, 32]`,最终的区间为`(22, 32]`|非唯一索引双端查询|
|`select * from t where c3 > 22 and c3 < 32 for update`|对于左区间，加锁范围为`(22, +∞]`，对于右区间，加锁范围为`(-∞, 32]`，最终的区间为`(22, 32]`|非唯一索引双端查询|
|`select * from t where c3 > 22 and c3 <= 32 for update`|对于左区间，加锁范围为`(22, +∞]`，对于右区间，加锁范围为`(-∞, 42]`，最终的区间为`(22, 42]`|非唯一索引双端查询|
|`select * from t where c3 >= 22 and c3 < 32 for update`|对于左区间，加锁范围为`(12, +∞]`，对于右区间，加锁范围为`(-∞，32]`，最终区间为`(12, 32]`|非唯一索引双端查询|
|`select * from t where c3 >= 22 and c3 <= 32 for update`|对于左区间，加锁范围为`(12, +∞]`，对于右区间，加锁范围为`(-∞, 42]`，最终区间为`(12, 42]`|非唯一索引双端查询|

### 查询条件是唯一索引

|SQL示例|加锁情况|类别|
|:---:|:---:|:---:|
|`select * from t where c2 = 21 for update`|初始时刻给c2=21上临键锁，即`LOCK_X|LOCK_ORDINARY`，加锁区间为`(12, 21]`，根据Principle 2，会继续向后寻找第一个不满足条件的记录，即c2=31，加上临键锁，加锁区间为`(21, 31]`，汇总后的加锁区间为`(11, 31]`，根据Principle 3，退化成间隙锁，即`LOCK_X|LOCK_GAP`，加锁区间为`(11, 31)`，根据Principle 3，唯一索引等值查询成功，退化成行锁，最后的加锁区间为`[21, 21]`|唯一索引等值查询成功|
|`select * from t where c2 = 20 for update`|c2=20记录不存在，向后寻找第一个满足条件的值，即c2=21，加上临键锁，加锁区间为`(11, 21]`，又根据Principle 3，退化成间隙锁，加锁区间为`(11, 21)`|唯一索引等值查询失败|
|`select * from t where c2 = 100 for update`|c2=100超过了表中的最大值，默认上个临键锁，加锁区间`(41, 100]`，根据Principle 2，会继续向后查找到`+∞`，即锁定`(100, +∞]`的区间，但是c2是唯一索引，超过返回不会加锁，因此最终没有数据加锁|唯一索引等值等值失败，查询值超过上限|
|`select * from t where c2 > 10 for update`|c2=10不存在，向后查找第一个值，即c2=11，加上临键锁，加锁范围`(-∞, 11]`，根据Principle 3退化成间隙锁，又由于没有右区间，合并加锁范围是`(-∞， +∞]`|唯一索引范围查询|唯一索引范围查询|
|`select * from t where c2 < 20 for update`|初始时刻给c2=21加上临键锁，加锁范围`(11, 21]`，根据Principle 3，退化成间隙锁，又因为没有左区间，因此最终加锁范围`(-∞, 21)`|唯一索引范围查询|