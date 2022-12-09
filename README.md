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
go get github.com/satori/go.uuid
go get github.com/panjf2000/ants/v2
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
redis-cli -a gf123456  --cluster rebalance 192.168.3.28:7006  --cluster-use-empty-masters 
(集成到shell脚本，在docekr中运行sh start.sh)

(3)测试业务
go test -benchmem -bench="InRecordParallel|LastPriceParallel|OutRecordParallel"  -benchtime=10s
```
### 缩容时测试
```
(1)用于获取node的id
redis-cli -a gf123456 --cluster check 192.168.3.28:7006
(2)自动迁移数据
redis-cli -a gf123456 --cluster reshard 192.168.3.28:7006 \
--cluster-from <node_id> \
--cluster-to  <node_id> \
--cluster-slots 4096 \
--cluster-yes
(3)测试业务
go test -benchmem -bench="InRecordParallel|LastPriceParallel|OutRecordParallel"  -benchtime=10s

go test -benchmem -bench="InRecordParallel|LastPriceParallel|OutRecordParallel"  -benchtime=18000s
```

### 查看pprof
```
cd pprof
go tool pprof -http=":8081"  cpu_profile
```

### 工具
```
# 清除数据
go run redisTool.go -clear  
```