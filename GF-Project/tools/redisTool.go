package main

import (
	"context"
	"flag"
	"log"
	"os"

	redis "github.com/go-redis/redis/v8"
)

var (
	ctx          = context.Background()
	clusterAddrs = []string{"192.168.3.28:7000"}
	password     = "gf123456"
	client, _    = NewGoRedisClient()
	clear        bool
)

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
		// var cursor uint64
		iter := client.Scan(ctx, 0, "*", 10000).Iterator()
		for iter.Next(ctx) {
			if err := client.Del(ctx, iter.Val()).Err(); err != nil {
				log.Println(err)
				panic(err)
			}
			log.Print(iter.Val())
		}
		iter = client.Scan(ctx, 0, "prefix:*", 10000).Iterator()
		for iter.Next(ctx) {
			if err := client.Del(ctx, iter.Val()).Err(); err != nil {
				log.Println(err)
				panic(err)
			}
			log.Print(iter.Val())
		}
		iter = client.SScan(ctx, "set-key", 0, "prefix:*", 10000).Iterator()
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
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	log.Println("begin")

	flag.BoolVar(&clear, "clear", false, "清除数据")
	flag.Parse()

	if clear {
		log.Println("clear")
		clearAllKey(client)
	}
	defer client.Close()
}
