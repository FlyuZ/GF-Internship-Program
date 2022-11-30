// Copyright (C) 2022
// Author zfy <522893161@qq.com>
// Build on 2022/11
package gf

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"strconv"
	"testing"
	"time"

	// "github.com/panjf2000/ants/v2"
)


var (
	file       *os.File
	cpuProfile *os.File
	memProfile *os.File
)

const (
	RunTimes      = 1000
	FakeDataLenth = 10000
)
//初始化
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

// 保留两位小数
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
			Client_id:       random_client_id[rand.Intn(len(random_client_id))],
			Exchange_type:   exchange_type_temp,
			Stock_code:      stock_code_temp,
			Entrust_bs:      "1",
			Business_amount: 100*rand.Int63n(9) + 100, //最少100手
		}
	}
	return fake_stockInfo
}

// 生成虚拟卖出时，查询redis里能卖的股票
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
				Client_id:       cur_client_id,
				Exchange_type:   exchange_type_temp,
				Stock_code:      cur_stock_code,
				Entrust_bs:      "2",
				Business_amount: 100,
			}
			index++
		}
	}
	for i := index; i < FakeDataLenth; i++ {
		exchange_type_temp, stock_code_temp := generateStockCode()
		fake_stockInfo[i] = StockInfo{
			Client_id:       random_client_id[rand.Intn(len(random_client_id))],
			Exchange_type:   exchange_type_temp,
			Stock_code:      stock_code_temp,
			Entrust_bs:      "2",
			Business_amount: 100,
		}
	}
	return fake_stockInfo
}

// 生成虚拟成交记录消息
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

func BenchmarkInRecordSerial(b *testing.B) {
	initlog()
	fake_stockInfo := GenerateStockRecord("1")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := RecordDistributionS(fake_stockInfo[i])
		log.Println(err)
	}
	b.StopTimer()
	close()
}

func BenchmarkLastPriceSerial(b *testing.B) {
	initlog()
	fake_lastPriceInfo := GenerateLastPrice()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := LastPriceDistributionS(fake_lastPriceInfo[i])
		log.Println(err)
	}
	b.StopTimer()
	close()
}

func BenchmarkOutRecordSerial(b *testing.B) {
	initlog()
	fake_stockInfo := GenerateStockRecord("2")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := RecordDistributionS(fake_stockInfo[i])
		log.Println(err)
	}
	b.StopTimer()
	close()
}

func BenchmarkInRecordParallel(b *testing.B) {
	initlog()
	var lockerr = errors.New("Lockfailed")
	fake_stockInfo := GenerateStockRecord("1")
	initppof() // 用于查看测试效果
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		go func(index int) {
			for {
				err := RecordDistributionP(fake_stockInfo[index])
				log.Println(err)
				if !errors.Is(err, lockerr) {
					break
				}
				time.Sleep(time.Duration(50) * time.Millisecond)
			}
		}(index)
		// 用于控制QPS
		time.Sleep(time.Duration(20) * time.Microsecond)
	}
	b.StopTimer()
	close()
}

func BenchmarkLastPriceParallel(b *testing.B) {
	initlog()
	var lockerr = errors.New("Lockfailed")
	fake_lastPriceInfo := GenerateLastPrice()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		go func(index int) {
			for {
				err := LastPriceDistributionP(fake_lastPriceInfo[index])
				log.Println(err)
				if !errors.Is(err, lockerr) {
					break
				}
				time.Sleep(time.Duration(50) * time.Millisecond)
			}
		}(index)
		time.Sleep(time.Duration(20) * time.Microsecond)
	}
	b.StopTimer()
	close()
}

func BenchmarkOutRecordParallel(b *testing.B) {
	initlog()
	var lockerr = errors.New("Lockfailed")
	fake_stockInfo := GenerateStockRecord("2")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		go func(index int) {
			for {
				err := RecordDistributionP(fake_stockInfo[index])
				log.Println(err)
				if !errors.Is(err, lockerr) {
					break
				}
				time.Sleep(time.Duration(50) * time.Millisecond)
			}
		}(index)
		time.Sleep(time.Duration(20) * time.Microsecond)
	}
	b.StopTimer()
	close()
}
