// Copyright (C) 2022
// Author zfy <522893161@qq.com>
// Build on 2022/11

package gf

import (
	"encoding/json"
	"log"
	"strings"
)

// 成交记录消息分发中心
func RecordDistributionS(stock_info StockInfo) error {
	// check stock_code set
	// if set not exists => create newholding
	// if entrust_bs == 1 => HoldingAddedUpdate
	// if entrust_bs == 2 => HoldingReductionUpdate
	stock_code_set, err := client.SIsMember(ctx, stock_info.Client_id, stock_info.Stock_code).Result()
	if err != nil {
		return err
	}
	if !stock_code_set && stock_info.Entrust_bs == "1" {
		newHoldingS(stock_info)
	} else if stock_code_set {
		client_holding_key := strings.Join([]string{stock_info.Client_id, stock_info.Stock_code}, "_")
		holding_info_str, _ := client.Get(ctx, client_holding_key).Result()
		var holding_info HoldingInfo
		json.Unmarshal([]byte(holding_info_str), &holding_info)
		if stock_info.Entrust_bs == "1" {
			holdingAddedUpdateS(stock_info, holding_info, client_holding_key)
		} else if stock_info.Entrust_bs == "2" {
			holdingReductionUpdateS(stock_info, holding_info, client_holding_key)
		}
	}
	return nil
}

// 最新价消息分发中心
func LastPriceDistributionS(lastPrice_info LastPriceInfo) error {
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
		latestPriceUpdateS(client_id, lastPrice_info)
	}
	return nil
}

// 新建持仓
func newHoldingS(stock_info StockInfo) error {
	client.SAdd(ctx, stock_info.Client_id, stock_info.Stock_code)
	client.LPush(ctx, stock_info.Stock_code, stock_info.Client_id)
	client_holding_key := strings.Join([]string{stock_info.Client_id, stock_info.Stock_code}, "_")
	// 价格应该是从最新价消息中获取
	holding_info := HoldingInfo{stock_info.Client_id, stock_info.Exchange_type, stock_info.Stock_code, stock_info.Business_amount, 0, 0}
	holding_info_str, err := json.Marshal(holding_info)
	client.Set(ctx, client_holding_key, holding_info_str, 0)
	return err
}

// 持仓新增更新
func holdingAddedUpdateS(stock_info StockInfo, holding_info HoldingInfo, client_holding_key string) error {
	holding_info.Hold_amount += stock_info.Business_amount
	holding_info.Market_value = float64(holding_info.Hold_amount) * holding_info.Last_price
	new_holding_info_str, err := json.Marshal(holding_info)
	client.Set(ctx, client_holding_key, new_holding_info_str, 0)
	return err
}

// 持仓减少更新/清空持仓
func holdingReductionUpdateS(stock_info StockInfo, holding_info HoldingInfo, client_holding_key string) error {
	if holding_info.Hold_amount > stock_info.Business_amount {
		holding_info.Hold_amount -= stock_info.Business_amount
		holding_info.Market_value = float64(holding_info.Hold_amount) * holding_info.Last_price
		new_holding_info_str, err := json.Marshal(holding_info)
		if err != nil {
			log.Println(err)
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
func latestPriceUpdateS(client_id string, lastPrice_info LastPriceInfo) error {
	client_holding_key := strings.Join([]string{client_id, lastPrice_info.Stock_code}, "_")
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
	return nil
}

func CloseS() {
	client.Close()
}
