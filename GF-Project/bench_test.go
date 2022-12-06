// Copyright (C) 2022
// Author zfy <522893161@qq.com>
// Build on 2022/11
package gf

import (
	"fmt"
	"log"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"strconv"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	// "github.com/panjf2000/ants/v2"
)

// 市场 (exchange_type) 为"1" (沪市)，"2”(深市)
// 沪市的stock_code 取值范围为:60000-600999,深市的stok code值范围为: 000001-001000，即沪深各1000个市场代码，last_price值返回为10.00-1000.00之间的随机数（两位小数）
// 客户号(client_id) 取值返回为:  000000000001-999999999999，随机获取100个客户号，每个客户号有1-100条持仓记录

var (
	file       *os.File
	cpuProfile *os.File
	memProfile *os.File
)

const (
	RunTimes      = 1000
	FakeDataLenth = 10000
	SleepTimes    = 15
)

func initlog() {
	file, _ = os.OpenFile("logfile.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	log.Println("load json success")
	client, _ = NewGoRedisClient()
}
func initppof() {
	cpuProfile, _ = os.Create("./pprof/cpu_profile")
	memProfile, _ = os.Create("./pprof/mem_profile")
	pprof.StartCPUProfile(cpuProfile)
	pprof.WriteHeapProfile(memProfile)
}
func close() {
	CloseP()
	pprof.StopCPUProfile()
	cpuProfile.Close()
	memProfile.Close()
	file.Close()
}

func Decimal(num float64) float64 {
	num, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", num), 64)
	return num
}

func generateStockCode() (string, string) {
	rand_string := []string{"1", "2"}
	rand.Seed(time.Now().UnixNano())
	exchange_type_temp := rand_string[rand.Intn(2)]
	var stock_code_temp string
	if exchange_type_temp == "1" {
		stock_code_temp = strconv.Itoa(rand.Intn(1000) + 600000)
	} else {
		stock_code_temp = strconv.Itoa(rand.Intn(1000) + 100000)
		stock_code_temp_byte := []byte(stock_code_temp)
		stock_code_temp_byte[0] = '0'
		stock_code_temp = string(stock_code_temp_byte)
	}
	return exchange_type_temp, stock_code_temp
}

func generateInStockRecord() [FakeDataLenth]StockInfo {
	var fake_stockInfo [FakeDataLenth]StockInfo
	for i := 0; i < FakeDataLenth; i++ {
		exchange_type_temp, stock_code_temp := generateStockCode()
		fake_stockInfo[i] = StockInfo{
			Order_id:        uuid.NewV4().String(),
			Client_id:       random_client_id[rand.Intn(len(random_client_id))],
			Exchange_type:   exchange_type_temp,
			Stock_code:      stock_code_temp,
			Entrust_bs:      "1",
			Business_amount: 100*rand.Int63n(9) + 100, //最少100手
		}
	}
	return fake_stockInfo
}

func generateOutStockRecord() [FakeDataLenth]StockInfo {
	var fake_stockInfo [FakeDataLenth]StockInfo
	index := 0
	for _, cur_client_id := range random_client_id {
		stock_code, _ := client.SMembers(ctx, cur_client_id).Result()
		for _, cur_stock_code := range stock_code {
			var exchange_type_temp string
			if cur_stock_code[0] == '6' {
				exchange_type_temp = "1"
			} else {
				exchange_type_temp = "2"
			}
			fake_stockInfo[index] = StockInfo{
				Order_id:        uuid.NewV4().String(),
				Client_id:       cur_client_id,
				Exchange_type:   exchange_type_temp,
				Stock_code:      cur_stock_code,
				Entrust_bs:      "2",
				Business_amount: 100*rand.Int63n(9) + 100, //最少100手
			}
			index++
			if index == FakeDataLenth {
				return fake_stockInfo
			}
		}
	}
	for i := index; i < FakeDataLenth; i++ {
		exchange_type_temp, stock_code_temp := generateStockCode()
		fake_stockInfo[i] = StockInfo{
			Order_id:        uuid.NewV4().String(),
			Client_id:       random_client_id[rand.Intn(len(random_client_id))],
			Exchange_type:   exchange_type_temp,
			Stock_code:      stock_code_temp,
			Entrust_bs:      "2",
			Business_amount: 100*rand.Int63n(9) + 100, //最少100手
		}
	}
	return fake_stockInfo
}

// 生成虚拟成交记录消息
// 调整一下卖出 都卖已有数据
func GenerateStockRecord(entrust_bs string) [FakeDataLenth]StockInfo {
	if entrust_bs == "1" {
		return generateInStockRecord()
	} else {
		return generateOutStockRecord()
	}
}

// 生成虚拟最新价消息
func GenerateLastPrice() [FakeDataLenth]LastPriceInfo {
	var fake_lastPriceInfo [FakeDataLenth]LastPriceInfo
	for i := 0; i < FakeDataLenth; i++ {
		exchange_type_temp, stock_code_temp := generateStockCode()
		fake_lastPriceInfo[i] = LastPriceInfo{
			Exchange_type: exchange_type_temp,
			Stock_code:    stock_code_temp,
			Last_price:    Decimal(10.00 + rand.Float64()*990.00),
		}
	}
	return fake_lastPriceInfo
}

func Benchmark_InRecordSerial(b *testing.B) {
	initlog()
	fake_stockInfo := GenerateStockRecord("1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		err := RecordDistributionS(fake_stockInfo[index])
		log.Println(err)
	}
	b.StopTimer()
	close()
}

func Benchmark_LastPriceSerial(b *testing.B) {
	initlog()
	fake_lastPriceInfo := GenerateLastPrice()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		err := LastPriceDistributionS(fake_lastPriceInfo[index])
		log.Println(err)
	}
	b.StopTimer()
	close()
}

func Benchmark_OutRecordSerial(b *testing.B) {
	initlog()
	fake_stockInfo := GenerateStockRecord("2")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		err := RecordDistributionS(fake_stockInfo[index])
		log.Println(err)
	}
	b.StopTimer()
	close()
}

func Benchmark_InRecordParallel(b *testing.B) {
	initlog()
	fake_stockInfo := GenerateStockRecord("1")
	initppof() // 用于查看测试效果
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		go func(index int) {
			for {
				err := RecordDistributionP(fake_stockInfo[index])
				if err == nil {
					break
				} else {
					log.Println(err)
				}
				time.Sleep(time.Duration(SleepTimes) * time.Millisecond)
			}
		}(index)
		// 用于控制QPS
		time.Sleep(time.Duration(SleepTimes) * time.Microsecond)
	}
	b.StopTimer()
	close()
}

func Benchmark_LastPriceParallel(b *testing.B) {
	initlog()
	fake_lastPriceInfo := GenerateLastPrice()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		go func(index int) {
			for {
				err := LastPriceDistributionP(fake_lastPriceInfo[index])
				if err == nil {
					break
				} else {
					log.Println(err)
				}
				time.Sleep(time.Duration(SleepTimes) * time.Millisecond)
			}
		}(index)
		time.Sleep(time.Duration(SleepTimes) * time.Microsecond)
	}
	b.StopTimer()
	close()
}

func Benchmark_OutRecordParallel(b *testing.B) {
	initlog()
	fake_stockInfo := GenerateStockRecord("2")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		go func(index int) {
			for {
				err := RecordDistributionP(fake_stockInfo[index])
				log.Println(err)
				if err == nil {
					break
				} else {
					log.Println(err)
				}
				time.Sleep(time.Duration(SleepTimes) * time.Millisecond)
			}
		}(index)
		time.Sleep(time.Duration(SleepTimes) * time.Microsecond)
	}
	b.StopTimer()
	close()
}
