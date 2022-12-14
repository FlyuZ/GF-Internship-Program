// Copyright (C) 2022
// Author zfy <522893161@qq.com>
// Build on 2022/11

package gf

import (
	"context"
	"log"
	"strconv"
	"time"
	"math/rand"
	"fmt"

	uuid "github.com/satori/go.uuid"

	redis "github.com/go-redis/redis/v8"
)

const FakeDataLenth = 10000
var (
	ctx          = context.Background()
	clusterAddrs = []string{"192.168.3.28:7001", "192.168.3.28:7003", "192.168.3.28:7005"}
	password     = "gf123456"
	client       *redis.ClusterClient
	// wg sync.WaitGroup
	random_client_id = []string{
		"691480561193", "991901594226", "093228024753", "902152659980", "677539460009", "158293574822", "682861270949", "149634593539", "866436173610", "432261066541", "212043813049", "555340979404", "229093876984", "757721061178", "021705950471", "677601744223", "320679218212", "432705818647", "158986102216", "727987584486", "959719516777", "738152028280", "438185167465", "942054030284", "164162533276", "422073291599", "261287046489", "948042040587", "807106729719", "109083208696", "240057665731", "676625786905", "054788425102", "800744952758", "234139769491", "774244770615", "911482242191", "963051459517", "837882905622", "522930089561", "032209849986", "318558780247", "954399933272", "018884707363", "710926541969", "773874727349", "850917991542", "312542849347", "664296036440", "933102703497", "207455500413", "951223969347", "471398922276", "487230332706", "481883259847", "781063140566", "999790902549", "544583671353", "102482760981", "759740164301", "720322457329", "972272194147", "047151868125", "031035838895", "150503338862", "876992565545", "627036779127", "943833980711", "706936910227", "515126938809", "584922526414", "962781591280", "781286848133", "390408705650", "877300451840", "534285127882", "845874812003", "844905767882", "520883036388", "983766961571", "688061611169", "307340303882", "127439828511", "702896747122", "321107842607", "415710152745", "617309269118", "005783939929", "334694122789", "756099419084", "568412646920", "252449862577", "858516941259", "479976108607", "324559964857", "849000874207", "385145050623", "964375937143", "827691120722", "860050833263"}
)

func NewGoRedisClient() (*redis.ClusterClient, error) {
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    clusterAddrs,
		Password: password,
	})
	res, err := client.Ping(ctx).Result()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	log.Println(res)
	return client, nil
}

// ????????????????????????
type LastPriceInfo struct {
	Exchange_type string  // ??????
	Stock_code    string  //??????
	Last_price    float64 // ?????????
}

// ???????????????????????????
type StockInfo struct {
	Order_id        string //????????????
	Client_id       string //?????????
	Exchange_type   string //??????
	Stock_code      string //??????
	Entrust_bs      string //????????????"1":?????????"2":??????
	Business_amount int64  //????????????
}

// ?????????????????????
type HoldingInfo struct {
	Client_id     string  //?????????
	Exchange_type string  //??????
	Stock_code    string  //??????
	Hold_amount   int64   //????????????
	Last_price    float64 //?????????
	Market_value  float64 //??????hold_amount*last_price
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
			Business_amount: 100*rand.Int63n(9) + 100, //??????100???
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
				Business_amount: 100*rand.Int63n(9) + 100, //??????100???
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
			Business_amount: 100*rand.Int63n(9) + 100, //??????100???
		}
	}
	return fake_stockInfo
}

// ??????????????????????????????
// ?????????????????? ??????????????????
func GenerateStockRecord(entrust_bs string) [FakeDataLenth]StockInfo {
	if entrust_bs == "1" {
		return generateInStockRecord()
	} else {
		return generateOutStockRecord()
	}
}

// ???????????????????????????
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