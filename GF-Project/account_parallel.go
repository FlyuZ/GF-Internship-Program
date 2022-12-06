// Copyright (C) 2022
// Author zfy <522893161@qq.com>
// Build on 2022/11

package gf

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"
)

func reidsLock(client_holding_key string) (bool, string) {
	client_holding_lock_key := strings.Join([]string{client_holding_key, "lock"}, "_")
	ok := client.SetNX(ctx, client_holding_lock_key, client_holding_key, time.Second*1).Val()
	return ok, client_holding_lock_key
}

// 成交记录消息分发中心
func RecordDistributionP(stock_info StockInfo) error {
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
	ok := client.SetNX(ctx, stock_info.Order_id, client_holding_key, time.Second*1).Val()
	if !ok {
		return nil
	}
	ok, client_holding_lock_key := reidsLock(client_holding_key)
	// 如果失败. 返回重试
	if !ok {
		return errors.New("Lockfailed")
	}
	if !stock_code_set && stock_info.Entrust_bs == "1" {
		newHoldingP(stock_info)
	} else if stock_code_set {
		holding_info_str, _ := client.Get(ctx, client_holding_key).Result()
		var holding_info HoldingInfo
		json.Unmarshal([]byte(holding_info_str), &holding_info)
		if stock_info.Entrust_bs == "1" {
			holdingAddedUpdateP(stock_info, holding_info, client_holding_key)
		} else if stock_info.Entrust_bs == "2" {
			holdingReductionUpdateP(stock_info, holding_info, client_holding_key)
		}
	}
	// 解锁
	client.Del(ctx, client_holding_lock_key)
	return nil
}

// 最新价消息分发中心
func LastPriceDistributionP(lastPrice_info LastPriceInfo) error {
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
		go func() {
			err = latestPriceUpdateP(client_id, lastPrice_info)
		}()
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}

// 新建持仓
func newHoldingP(stock_info StockInfo) error {
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
func holdingAddedUpdateP(stock_info StockInfo, holding_info HoldingInfo, client_holding_key string) error {
	holding_info.Hold_amount += stock_info.Business_amount
	holding_info.Market_value = float64(holding_info.Hold_amount) * holding_info.Last_price
	new_holding_info_str, err := json.Marshal(holding_info)
	client.Set(ctx, client_holding_key, new_holding_info_str, 0)
	return err
}

// 持仓减少更新/清空持仓
func holdingReductionUpdateP(stock_info StockInfo, holding_info HoldingInfo, client_holding_key string) error {
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
func latestPriceUpdateP(client_id string, lastPrice_info LastPriceInfo) error {
	// 上锁
	client_holding_key := strings.Join([]string{client_id, lastPrice_info.Stock_code}, "_")
	ok, client_holding_lock_key := reidsLock(client_holding_key)
	if !ok {
		// log.Println("lock failed")
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

func CloseP() {
	client.Close()
}
