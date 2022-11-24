package gf

import (
	"log"
	"context"

	redis "github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var clusterAddrs = []string{"192.168.3.28:7000","192.168.3.28:7001","192.168.3.28:7002"}
var password     = "gf123456"

func NewGoRedisClient() (*redis.ClusterClient, error) {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    clusterAddrs,
		Password: password,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		log.Println(err)
		return nil, err
	}
	return client, nil
}
