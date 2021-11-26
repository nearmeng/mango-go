# bingo redis db支持


## 以tcaplus为基准实现初版，会修改部分simple接口

- Key为FullName:{key-str},多key用-连接,例如TBAcntInfo:{1123-xxx},key-str必须写在hashtag内保证集群指向正常
- 带有version和不带version版本redis的读写操作**不走同一指令(脚本)**，主要是性能考虑，有的时候不需要版本号的时候不值得浪费性能，讨论过后决定从simple接口中删除版本号支持
- 第一版本暂时不支持额外索引,只支持主key查询,后续实现方案也很简单，增加_link字段
- ~~有版本号和无版本号混合使用必须要知道会产生什么样的后果。。潜规则不推荐使用.**最好使用有版本号的接口就不要使用不带版本号的,反之亦然**,不过可以尝试一下尽力支持兼容，可能有些地方可以提供一些优化手段。~~
- 所有数据放在一个字段的做法过于简陋粗暴，直接实现分散存储到hash中，增加_ver等控制字段+lua脚本来实现类似tcaplus的功能
- List 的支持增加一个同名(同前缀)meta hash/string
- 目前有2个方案实现wireformat数据分散存储
    - ~~marshalToMap， 需要public一个私有方法 MarshalOptions.marshalField ,尝试了一下放弃了，为这这个去patch proto库的实现不是很划算~~
    - 直接protowire.ConsumeTag读取wiretag和length来将一个bytes分拆到map，使用fieldNumber作为key要方便一些
        - fieldNumber作为key， 最好dirty fields也用fieldnumber来指定， 但是现在是用string来标记的.
        - 用string来作为key ，可读性好一点，但是跟pb使用fieldnumber来区别字段逻辑不一样.
        - 类int(数字)类型使用wire来存取还是以intstring来存取，使用wireformat存储无法自己inc，但是使用intstring来存取会造成打包解包复杂化， 都使用过，不推荐普通db存储类型使用inc
        - 暂时方案跟tcaplus兼容使用fieldName string作为hashkey
    - 不考虑redis-db直接支持protobuf的任何功能，因为不是所有的redis实现和集群方案都支持,所有支持都由bingo的dbplugin支持完成


## 目前进度
- [x] 查改增删基础支持
- [ ] 查改增删version支持
- [ ] increase支持
- [ ] list支持
- [ ] list带version支持
- [ ] 测试代码整理覆盖