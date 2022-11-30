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
go test -benchmem -bench=.  -benchtime=1s
```

### 查看pprof
```
cd pprof
go tool pprof -http=":8081"  cpu_profile
```