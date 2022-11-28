// Copyright (C) 2022
// Author zfy <522893161@qq.com>
// Build on 2022/11
package gf

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"runtime/pprof"
	_ "net/http/pprof"
	"testing"
	"time"
	// "flag"

	"github.com/panjf2000/ants/v2"
)

// 市场 (exchange_type) 为"1" (沪市)，"2”(深市)
// 沪市的stock_code 取值范围为:60000-600999,深市的stok code值范围为: 000001-001000，即沪深各1000个市场代码，last_price值返回为10.00-1000.00之间的随机数（两位小数）
// 客户号(client_id) 取值返回为:  000000000001-999999999999，随机获取100个客户号，每个客户号有1-100条持仓记录

var (
	file *os.File
	cpuProfile *os.File
	memProfile *os.File
)

const (
	RunTimes = 1000
	FakeDataLenth = 10000
)

func initlog() {
	file, _ = os.OpenFile("logfile.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	log.Println("load json success")
}
func initppof(){
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

// 生成虚拟成交记录消息
func GenerateStockRecord(entrust_bs string) [FakeDataLenth]StockInfo {
	var fake_stockInfo [FakeDataLenth]StockInfo
	for i := 0; i < FakeDataLenth; i++ {
		exchange_type_temp, stock_code_temp := generateStockCode()
		fake_stockInfo[i] = StockInfo{
			Client_id:       random_client_id[rand.Intn(len(random_client_id))],
			Exchange_type:   exchange_type_temp,
			Stock_code:      stock_code_temp,
			Entrust_bs:      entrust_bs,
			Business_amount: 100*rand.Int63n(9) + 100, //最少100手
		}
	}
	return fake_stockInfo
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
	b.ResetTimer() // 重置定时器
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
	b.ResetTimer() // 重置定时器
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
	b.ResetTimer() // 重置定时器
	for i := 0; i < b.N; i++ {
		err := RecordDistributionS(fake_stockInfo[i])
		log.Println(err)
	}
	b.StopTimer()
	close()
}

func BenchmarkInRecordParallel(b *testing.B) {
	initlog()
	defer ants.Release()
	var lockerr = errors.New("lock error")
	fake_stockInfo := GenerateStockRecord("1")
	initppof()
	b.ResetTimer() // 重置定时器
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		go func(index int) {
			for {
				err := RecordDistributionP(fake_stockInfo[index])
				log.Println(err)
				log.Println(index)
				if !errors.Is(err, lockerr) {
					break
				}
			}
		}(index)
	}
	b.StopTimer()
	close()
}
func BenchmarkLastPriceParallel(b *testing.B) {
	initlog()
	defer ants.Release()
	var lockerr = errors.New("lock error")
	fake_lastPriceInfo := GenerateLastPrice()
	b.ResetTimer() // 重置定时器
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		go func(index int) {
			for {
				err := LastPriceDistributionP(fake_lastPriceInfo[index])
				log.Println(err)
				log.Println(index)
				if !errors.Is(err, lockerr) {
					break
				}
			}
		}(index)
	}
	b.StopTimer()
	close()
}

func BenchmarkOutRecordParallel(b *testing.B) {
	initlog()
	defer ants.Release()
	var lockerr = errors.New("lock error")
	fake_stockInfo := GenerateStockRecord("2")
	b.ResetTimer() // 重置定时器
	for i := 0; i < b.N; i++ {
		index := rand.Intn(10000)
		go func(index int) {
			for {
				err := RecordDistributionP(fake_stockInfo[index])
				log.Println(err)
				log.Println(index)
				if !errors.Is(err, lockerr) {
					break
				}
			}
		}(index)
	}
	b.StopTimer()
	close()
}
