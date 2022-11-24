package gf

import (
	"encoding/json"
	"log"
	"strings"
)

// var lock = &sync.Mutex{} //创建互锁
// 最新价消息结构体
var client, _ = NewGoRedisClient()

type lastPriceInfo struct {
	Exchange_type string  // 市场
	Stock_code    string  //代码
	Last_price    float64 // 最新价
}

// 成交记录消息结构体
type stockInfo struct {
	Client_id       string //客户号
	Exchange_type   string //市场
	Stock_code      string //代码
	Entrust_bs      string //买卖方向"1":买入、"2":卖出
	Business_amount int64  //成交数量
}

// 持仓消息结构体
type holdingInfo struct {
	Client_id     string  //客户号
	Exchange_type string  //市场
	Stock_code    string  //代码
	Hold_amount   int64   //持仓数量
	Last_price    float64 //最新价
	Market_value  float64 //市值hold_amount*last_price
}

// 成交记录消息分发中心
func RecordDistribution(stock_info stockInfo) error {
	// check stock_code set
	// if set not exists => create newholding
	// if entrust_bs == 1 => HoldingAddedUpdate
	// if entrust_bs == 2 => HoldingReductionUpdate
	stock_code_set, err := client.SIsMember(ctx, stock_info.Client_id, stock_info.Stock_code).Result()
	if err != nil {
		return err
	}
	if !stock_code_set {
		NewHolding(stock_info)
	} else {
		client_holding_key := strings.Join([]string{stock_info.Client_id, stock_info.Stock_code}, "_")
		holding_info_str, _ := client.Get(ctx, client_holding_key).Result()
		var holding_info holdingInfo
		json.Unmarshal([]byte(holding_info_str), &holding_info)
		if stock_info.Entrust_bs == "1" {
			HoldingAddedUpdate(stock_info, holding_info, client_holding_key)
		} else if stock_info.Entrust_bs == "2" {
			HoldingReductionUpdate(stock_info, holding_info, client_holding_key)
		}
	}
	return nil
}

// 最新价消息分发中心
func LatestPriceDistribution(lastPrice_info lastPriceInfo) error {
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
		LatestPriceUpdate(client_id, lastPrice_info)
	}
	return nil
}

// 新建持仓
func NewHolding(stock_info stockInfo) error {
	client.SAdd(ctx, stock_info.Client_id, stock_info.Stock_code)
	client.LPush(ctx, stock_info.Stock_code, stock_info.Client_id)
	client_holding_key := strings.Join([]string{stock_info.Client_id, stock_info.Stock_code}, "_")
	// 价格应该是从最新价消息中获取
	holding_info := holdingInfo{stock_info.Client_id, stock_info.Exchange_type, stock_info.Stock_code, stock_info.Business_amount, 0, 0}
	holding_info_str, err := json.Marshal(holding_info)
	client.Set(ctx, client_holding_key, holding_info_str, 0)
	return err
}

// 持仓新增更新
func HoldingAddedUpdate(stock_info stockInfo, holding_info holdingInfo, client_holding_key string) error {
	holding_info.Hold_amount += stock_info.Business_amount
	holding_info.Market_value = float64(holding_info.Hold_amount) * holding_info.Last_price
	new_holding_info_str, err := json.Marshal(holding_info)
	client.Set(ctx, client_holding_key, new_holding_info_str, 0)
	return err
}

// 持仓减少更新/清空持仓
func HoldingReductionUpdate(stock_info stockInfo, holding_info holdingInfo, client_holding_key string) error {
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
func LatestPriceUpdate(client_id string, lastPrice_info lastPriceInfo) error {
	client_holding_key := strings.Join([]string{client_id, lastPrice_info.Stock_code}, "_")
	holding_info_str, err := client.Get(ctx, client_holding_key).Result()
	if err != nil {
		return err
	}
	var holding_info holdingInfo
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
