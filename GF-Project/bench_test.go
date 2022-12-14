// Copyright (C) 2022
// Author zfy <522893161@qq.com>
// Build on 2022/11
package gf

import (
	"log"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"testing"
	"time"
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
	RunTimes = 1000
	// FakeDataLenth = 10000
	SleepTimes = 100
)

func initlog() {
	file, _ = os.OpenFile("d:\\logfile.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
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
					// log.Println("success")
					break
				} else {
					log.Println(err)
				}
				time.Sleep(time.Duration(SleepTimes) * time.Millisecond)
			}
		}(index)
		// 用于控制QPS
		time.Sleep(time.Duration(SleepTimes) * time.Millisecond)
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
					// log.Println("success")
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
					// log.Println("success")
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
