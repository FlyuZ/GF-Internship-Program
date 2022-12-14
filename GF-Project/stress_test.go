package gf

import (
	"log"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
)

func init_log() {
	file, _ := os.OpenFile("d:\\logfile.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, os.ModePerm)
	log.SetOutput(file)
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
	log.Println("load json success")
	client, _ = NewGoRedisClient()
}
func init_ppof() {
	cpuProfile, _ := os.Create("./pprof/cpu_profile")
	memProfile, _ := os.Create("./pprof/mem_profile")
	pprof.StartCPUProfile(cpuProfile)
	pprof.WriteHeapProfile(memProfile)
}
func Test_Stress(t *testing.T) {
	init_log()
	init_ppof()
	fake_stockInfo_1 := GenerateStockRecord("1")
	fake_lastPriceInfo := GenerateLastPrice()
	fake_stockInfo_2 := GenerateStockRecord("2")

	p, _ := ants.NewPool(10000)
	defer p.Release()
	for {
		p.Submit(func() {
			index := rand.Intn(10000)
			for {
				err := RecordDistributionS(fake_stockInfo_1[index])
				if err == nil {
					break
				}
				log.Println(err)
				time.Sleep(time.Duration(100) * time.Microsecond)
			}
		})
		p.Submit(func() {
			index := rand.Intn(10000)
			for {
				err := LastPriceDistributionS(fake_lastPriceInfo[index])
				if err == nil {
					break
				}
				log.Println(err)
				time.Sleep(time.Duration(100) * time.Microsecond)
			}
		})
		p.Submit(func() {
			index := rand.Intn(10000)
			for {
				err := RecordDistributionS(fake_stockInfo_2[index])
				if err == nil {
					break
				}
				log.Println(err)
				time.Sleep(time.Duration(100) * time.Microsecond)
			}
		})
		time.Sleep(time.Duration(1) * time.Millisecond)
	}
}
