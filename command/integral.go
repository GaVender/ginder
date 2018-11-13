package command

import (
	"ginder/conf"
	"fmt"
	"encoding/json"
	"time"
	"strconv"
	"os"
	"gopkg.in/redis.v5"
	"github.com/jmoiron/sqlx"
	"sync/atomic"
	"sync"
)

const DealIdRedisKEY  = "string:integral_expire_deal_id"
const DealIdRedisTime = 60 * 60 * 24 * 7
const DealNum		  = 10
const GoroutineNum	  = 10

var redisObj *redis.Client
var dbSlave *sqlx.DB
var dbMaster *sqlx.DB
var mx *sync.RWMutex
var wg *sync.WaitGroup

type integralExpireConfig struct {
	IsOpen 		uint8 `json:"is_open"`
	ExpireType 	uint8 `json:"expire_type"`
}

type userIntegralExpire struct {
	Id uint64 `db:"id" json:"id"`
	Uid uint64 `db:"uid" json:"uid"`
	Integral uint64 `db:"integral" json:"integral"`
}

type integralDetail struct {
	ID uint64 `json:"id" db:"id"`
	MemberId uint64 `json:"member_id" db:"member_id"`
	SurplusMabi float32 `json:"surplus_mabi" db:"surplus_mabi"`
	MabiSource float32 `json:"mabi_source" db:"mabi_source"`
	Type uint16 `json:"type" db:"type"`
	CreateTime string `json:"create_time" db:"create_time"`
	UpdateTime string `json:"update_time" db:"update_time"`
	SourceType uint8 `json:"source_type" db:"source_type"`
	RecordNo string `json:"record_no" db:"record_no"`
	IntegralCode string `json:"integral_code" db:"integral_code"`
	Remark string `json:"remark" db:"remark"`
	CardNo string `json:"card_no" db:"card_no"`
}

func init() {
	redisObj = conf.RedisMaster()
	dbSlave  = conf.SqlSlaveDb()
	dbMaster = conf.SqlMasterDb()
	mx = new(sync.RWMutex)
	wg = new(sync.WaitGroup)
}

func AutoClean() {
	defer dbMaster.Close()
	defer dbSlave.Close()

	integralConfig := getIntegralConfig()
	_, beginTime, endTime := isExpireDealTime(integralConfig.ExpireType)

	idRedis := getDealId()
	idMysql := getBigDealId()

	if idRedis >= idMysql {
		fmt.Println("脚本已清除积分完毕")
	} else {
		fmt.Println("总数是：", idMysql)

		if idRedis <= 0 {
			idRedis = 1
		}

		var beginId uint64

		for {
			idRedis = getDealId()

			if idRedis >= idMysql {
				break
			}

			atomic.StoreUint64(&beginId, idRedis)

			for i := 0; i < GoroutineNum; i ++ {
				wg.Add(1)
				go dealUserIntegral(beginId, beginTime, endTime)
				atomic.AddUint64(&beginId, DealNum)
			}

			wg.Wait()
			time.Sleep(time.Second * 2)
			break
		}
	}
}



func getIntegralConfig() *integralExpireConfig {
	var redisKey = "string:yunying:integral_expire_config"

	integralExpireConfigInfo := redisObj.Get(redisKey).Val()
	i := &integralExpireConfig{}
	err := json.Unmarshal([]byte(integralExpireConfigInfo), i)

	if err != nil {
		fmt.Println("积分配置获取出错")
		os.Exit(0)
	} else {
		fmt.Println("积分配置获取成功：", i)

		if i.IsOpen == 0 {
			fmt.Println("积分过期配置没开启")
			os.Exit(0)
		}
	}

	return i
}

func isExpireDealTime(expireType uint8) (bool, string, string) {
	nowMonth  := int(time.Now().Month())
	flag 	  := false
	beginTime := ""
	endTime   := ""

	if expireType == 1 {
		if nowMonth == 3 {
			flag = true
			beginTime = strconv.Itoa(time.Now().Year() - 1) + "-03-01 0:0:0"
			endTime = strconv.Itoa(time.Now().Year()) + "-03-01 0:0:0"
		}
	} else {
		if nowMonth == 11 {
			flag = true
			beginTime = strconv.Itoa(time.Now().Year() - 1) + "-01-01 0:0:0"
			endTime = strconv.Itoa(time.Now().Year()) + "-01-01 0:0:0"
		} else if nowMonth == 7 {
			flag = true
			beginTime = strconv.Itoa(time.Now().Year() - 1) + "-07-01 0:0:0"
			endTime = strconv.Itoa(time.Now().Year()) + "-07-01 0:0:0"
		}
	}

	if !flag {
		fmt.Println("还未到清除积分月份")
		os.Exit(0)
	}

	return flag, beginTime, endTime
}

func getDealId() uint64 {
	mx.RLock()
	defer mx.RUnlock()

	var v = redisObj.Get(DealIdRedisKEY).Val()

	if v == "" {
		return 0
	} else {
		k, _ := strconv.Atoi(v)
		return uint64(k)
	}
}

func setDealId(i uint64) {
	mx.Lock()
	defer mx.Unlock()

	redisObj.Set(DealIdRedisKEY, i, time.Second * DealIdRedisTime)
}

func getBigDealId() uint64 {
	u := userIntegralExpire{}
	err := dbSlave.Get(&u, "select id from finance.user_expire_integral order by id desc limit 1")

	if err != nil {
		fmt.Println("获取积分过期表的ID出错：", err.Error())
		os.Exit(0)
	}

	return u.Id
}

func dealUserIntegral(id uint64, expireBeginTime string, expireEndTime string) {
	defer wg.Done()

	beginId := id
	atomic.AddUint64(&id, DealNum)
	users := []userIntegralExpire{}
	err := dbSlave.Select(&users, "select uid, integral from finance.user_expire_integral where id >= ? and id < ?",
		beginId, id)

	if err != nil {
		fmt.Println("在处理用户过期积分出错：", err.Error())
	} else {
		count := len(users)
		fmt.Println("清理的开始ID：", beginId, "，清理的用户数量：", count)

		for _, u := range users {
			calUserUsedIntegral(u.Uid, expireBeginTime, expireEndTime)
		}
	}

	setDealId(id)
}

func calUserUsedIntegral(uid uint64, expireBeginTime string, expireEndTime string) {
	integralDetail := integralDetail{}
	sql := "select member_id, sum(mabi_source) from orders.integral_detail where member_id in (?) and mabi_source < 0 " +
		"and create_time < ? and create_time > ? group by member_id"
	err := dbSlave.Get(&integralDetail, sql, uid, expireEndTime, expireBeginTime)

	if err != nil {
		fmt.Println("获取用户积分出错：", err.Error())
	} else {
		fmt.Println(integralDetail)
	}
}