### 新建Redis-Cluster   docker-compose
```
docker-compose up -d
```

### 集群模式登录Redis服务器  -c  集群模式登录
```
docker exec -ti #{容器id} redis-cli -c -p #{Redis设置的端口号} 
```

### 配置
```
go get github.com/go-redis/redis/v8
```

### 测试
```
go test -benchmem -bench="."  -benchtime=1s
```

### 扩容时测试
```
(1)新建redis服务节点
docker-compose -f docker-compose-add.yml up -d

(2)将节点加入集群
docker exec -it redis-server-6 sh redis-cli -a gf123456 --cluster check 192.168.3.28:7006
redis-cli -a gf123456 --cluster add-node 192.168.3.28:7006 192.168.3.28:7001
redis-cli -a gf123456 --cluster reshard  192.168.3.28:7006
yes  all

(3)测试业务
go test -benchmem -bench="InRecordParallel|LastPriceParallel|OutRecordParallel"  -benchtime=3s
```
### 缩容时测试
```
需要先把预删除节点中的数据迁移
```

### 查看pprof
```
cd pprof
go tool pprof -http=":8081"  cpu_profile
```