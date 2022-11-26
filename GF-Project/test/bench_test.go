// Copyright (C) 2022
// Author zfy <522893161@qq.com>
// Build on 2022/11
package gf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	// account "github.com/FlyuZ/gf/account_serial"
	account "github.com/FlyuZ/gf/account_parallel"
)

// 市场 (exchange_type) 为"1" (沪市)，"2”(深市)
// 沪市的stock_code 取值范围为:60000-600999,深市的stok code值范围为: 000001-001000，即沪深各1000个市场代码，last_price值返回为10.00-1000.00之间的随机数（两位小数）
// 客户号(client_id) 取值返回为:  000000000001-999999999999，随机获取100个客户号，每个客户号有1-100条持仓记录
type randomData struct {
	Stock_code []string
	Client_id  []string
}

var random_data randomData

func Init() {
	var data, _ = ioutil.ReadFile("data.json")
	json.Unmarshal(data, &random_data)
	log.Println("load json success")
}

func Decimal(num float64) float64 {
	num, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", num), 64)
	return num
}

// 生成虚拟成交记录消息
func GenerateStockRecord() account.StockInfo {
	rand.Seed(time.Now().UnixNano())
	rand_string := []string{"1", "2"}
	fake_stockInfo := account.StockInfo{
		Client_id:       random_data.Client_id[rand.Intn(len(random_data.Client_id))],
		Exchange_type:   rand_string[rand.Intn(2)],
		Stock_code:      random_data.Stock_code[rand.Intn(len(random_data.Stock_code))],
		Entrust_bs:      rand_string[rand.Intn(2)],
		Business_amount: 100*rand.Int63n(9) + 100, //最少100手
	}
	return fake_stockInfo
}

// 生成虚拟最新价消息
func GenerateLastPrice() account.LastPriceInfo {
	rand.Seed(time.Now().UnixNano())
	rand_string := []string{"1", "2"}
	fake_lastPriceInfo := account.LastPriceInfo{
		Exchange_type: rand_string[rand.Intn(2)],
		Stock_code:    random_data.Stock_code[rand.Intn(len(random_data.Stock_code))],
		Last_price:    Decimal(10.00 + rand.Float64()*990.00),
	}
	return fake_lastPriceInfo
}

func TestRecordSerial(t *testing.T) {
	file, _ := os.OpenFile("logfile.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	defer file.Close()
	log.SetOutput(file)
	log.SetFlags(log.Llongfile)
	t.Run("go-redis", func(t *testing.T) {
		Init()
		for i := 0; i < 1000; i++ {
			fake_stockInfo := GenerateStockRecord()
			err := account.RecordDistribution(fake_stockInfo)
			log.Println(err)
			fake_lastPriceInfo := GenerateLastPrice()
			err = account.LastPriceDistribution(fake_lastPriceInfo)
			log.Println(err)
		}
		defer account.Close()
	})
}

func TestRecordParallel(t *testing.T) {
	file, _ := os.OpenFile("logfile.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	defer file.Close()
	log.SetOutput(file)
	log.SetFlags(log.Llongfile)
	Init()

	defer ants.Release()
	var wg sync.WaitGroup
	var lockerr = errors.New("lock error")
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		_ = ants.Submit(func() {
			defer wg.Done()
			fake_stockInfo := GenerateStockRecord()
			err := account.RecordDistribution(fake_stockInfo)
			if errors.Is(err, lockerr) {
				for j := 0; j < 3; j++ {
					err = account.RecordDistribution(fake_stockInfo)
				}
			}
			log.Println(err)
		})
		wg.Add(1)
		_ = ants.Submit(func() {
			defer wg.Done()
			fake_lastPriceInfo := GenerateLastPrice()
			err := account.LastPriceDistribution(fake_lastPriceInfo)
			if errors.Is(err, lockerr) {
				for j := 0; j < 3; j++ {
					err = account.LastPriceDistribution(fake_lastPriceInfo)
				}
			}
			log.Println(err)
		})
	}
	wg.Wait()
}
