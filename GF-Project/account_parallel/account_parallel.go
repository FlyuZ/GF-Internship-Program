// Copyright (C) 2022
// Author zfy <522893161@qq.com>
// Build on 2022/11

// 业务逻辑

package gf

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	redis "github.com/go-redis/redis/v8"
	"github.com/panjf2000/ants/v2"
)

var ctx = context.Background()
var clusterAddrs = []string{"192.168.3.28:7000", "192.168.3.28:7001", "192.168.3.28:7002"}
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

var client, _ = NewGoRedisClient()
var wg sync.WaitGroup

// 最新价消息结构体
type LastPriceInfo struct {
	Exchange_type string  // 市场
	Stock_code    string  //代码
	Last_price    float64 // 最新价
}

// 成交记录消息结构体
type StockInfo struct {
	Client_id       string //客户号
	Exchange_type   string //市场
	Stock_code      string //代码
	Entrust_bs      string //买卖方向"1":买入、"2":卖出
	Business_amount int64  //成交数量
}

// 持仓消息结构体
type HoldingInfo struct {
	Client_id     string  //客户号
	Exchange_type string  //市场
	Stock_code    string  //代码
	Hold_amount   int64   //持仓数量
	Last_price    float64 //最新价
	Market_value  float64 //市值hold_amount*last_price
}

func reidsLock(client_holding_key string) (bool, string) {
	random_value, _ := rand.Prime(rand.Reader, 9) // 随时生成9位随机数
	client_holding_lock_key := strings.Join([]string{client_holding_key, "lock"}, "_")
	ok := client.SetNX(ctx, client_holding_lock_key, random_value, time.Second*1)
	return ok.Val(), client_holding_lock_key
}

// 成交记录消息分发中心
func RecordDistribution(stock_info StockInfo) error {
	// check stock_code set
	// if set not exists => create newholding
	// if entrust_bs == 1 => HoldingAddedUpdate
	// if entrust_bs == 2 => HoldingReductionUpdate
	stock_code_set, err := client.SIsMember(ctx, stock_info.Client_id, stock_info.Stock_code).Result()
	if err != nil {
		return err
	}
	client_holding_key := strings.Join([]string{stock_info.Client_id, stock_info.Stock_code}, "_")
	// 上锁
	ok, client_holding_lock_key := reidsLock(client_holding_key)
	if !ok {
		return errors.New("lock failed")
	}
	if !stock_code_set {
		NewHolding(stock_info)
	} else {
		holding_info_str, _ := client.Get(ctx, client_holding_key).Result()
		var holding_info HoldingInfo
		json.Unmarshal([]byte(holding_info_str), &holding_info)
		if stock_info.Entrust_bs == "1" {
			HoldingAddedUpdate(stock_info, holding_info, client_holding_key)
		} else if stock_info.Entrust_bs == "2" {
			HoldingReductionUpdate(stock_info, holding_info, client_holding_key)
		}
	}
	// 解锁
	client.Del(ctx, client_holding_lock_key)
	return nil
}

// 最新价消息分发中心
func LastPriceDistribution(lastPrice_info LastPriceInfo) error {
	// get client_id_list
	client_id_llen, err := client.LLen(ctx, lastPrice_info.Stock_code).Result()
	if err != nil {
		return err
	}
	client_id_list, err := client.LRange(ctx, lastPrice_info.Stock_code, 0, client_id_llen).Result()
	if err != nil {
		return err
	}
	for _, client_id := range client_id_list {
		wg.Add(1)
		err = ants.Submit(func() {
			LatestPriceUpdate(client_id, lastPrice_info)
			wg.Done()
		})
		if err != nil {
			log.Println(err)
		}
	}
	wg.Wait()
	return nil
}

// 新建持仓
func NewHolding(stock_info StockInfo) error {
	client.SAdd(ctx, stock_info.Client_id, stock_info.Stock_code)
	client.LPush(ctx, stock_info.Stock_code, stock_info.Client_id)
	client_holding_key := strings.Join([]string{stock_info.Client_id, stock_info.Stock_code}, "_")
	// 价格默认是0
	holding_info := HoldingInfo{stock_info.Client_id, stock_info.Exchange_type, stock_info.Stock_code, stock_info.Business_amount, 0, 0}
	holding_info_str, err := json.Marshal(holding_info)
	client.Set(ctx, client_holding_key, holding_info_str, 0)
	return err
}

// 持仓新增更新
func HoldingAddedUpdate(stock_info StockInfo, holding_info HoldingInfo, client_holding_key string) error {
	holding_info.Hold_amount += stock_info.Business_amount
	holding_info.Market_value = float64(holding_info.Hold_amount) * holding_info.Last_price
	new_holding_info_str, err := json.Marshal(holding_info)
	client.Set(ctx, client_holding_key, new_holding_info_str, 0)
	return err
}

// 持仓减少更新/清空持仓
func HoldingReductionUpdate(stock_info StockInfo, holding_info HoldingInfo, client_holding_key string) error {
	if holding_info.Hold_amount > stock_info.Business_amount {
		holding_info.Hold_amount -= stock_info.Business_amount
		holding_info.Market_value = float64(holding_info.Hold_amount) * holding_info.Last_price
		new_holding_info_str, err := json.Marshal(holding_info)
		if err != nil {
			return err
		}
		client.Set(ctx, client_holding_key, new_holding_info_str, 0)
	} else {
		// 清空持仓
		client.Del(ctx, client_holding_key)
		client.SRem(ctx, stock_info.Stock_code, stock_info.Client_id)
	}
	return nil
}

// 持仓市值更新
func LatestPriceUpdate(client_id string, lastPrice_info LastPriceInfo) error {
	// 上锁
	client_holding_key := strings.Join([]string{client_id, lastPrice_info.Stock_code}, "_")
	ok, client_holding_lock_key := reidsLock(client_holding_key)
	if !ok {
		log.Println("lock failed")
		return errors.New("lock failed")
	}
	holding_info_str, err := client.Get(ctx, client_holding_key).Result()
	if err != nil {
		return err
	}
	var holding_info HoldingInfo
	json.Unmarshal([]byte(holding_info_str), &holding_info)
	holding_info.Last_price = lastPrice_info.Last_price
	holding_info.Market_value = float64(holding_info.Hold_amount) * holding_info.Last_price
	new_holding_info_str, err := json.Marshal(holding_info)
	if err != nil {
		return err
	}
	client.Set(ctx, client_holding_key, new_holding_info_str, 0)
	// 解锁
	client.Del(ctx, client_holding_lock_key)
	return nil
}

func Close() {
	ants.Release()
}
