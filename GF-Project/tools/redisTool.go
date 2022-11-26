package main

import (
	"context"
	"log"
	"os"

	redis "github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var clusterAddrs = []string{"192.168.3.28:7003"}
var password = "gf123456"

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

func clearAllKey(c *redis.ClusterClient) {
	err := c.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
		log.Println("client", client)
		var cursor uint64
		iter := client.Scan(ctx, cursor, "*", 10000).Iterator()
		for iter.Next(ctx) {
			if err := client.Del(ctx, iter.Val()).Err(); err != nil {
				log.Println(err)
				panic(err)
			}
			log.Print(iter.Val())
		}
		iter = client.Scan(ctx, cursor, "prefix:*", 10000).Iterator()
		for iter.Next(ctx) {
			if err := client.Del(ctx, iter.Val()).Err(); err != nil {
				log.Println(err)
				panic(err)
			}
			log.Print(iter.Val())
		}
		iter = client.SScan(ctx, "set-key", cursor, "prefix:*", 10000).Iterator()
		for iter.Next(ctx) {
			if err := client.SRem(ctx, "set-key", iter.Val()).Err(); err != nil {
				log.Println(err)
				panic(err)
			}
			log.Print(iter.Val())
		}
		return iter.Err()
	})
	if err != nil {
		panic(err)
	}
}


func main() {
	file, _ := os.OpenFile("logfile.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	defer file.Close()
	log.SetOutput(file)
	log.SetFlags(log.Llongfile)
	log.Println("begin")
	client, _ := NewGoRedisClient()
	clearAllKey(client)
	defer client.Close()
}
