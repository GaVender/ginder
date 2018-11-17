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
	"math/rand"
	"math"
)

const DealIdRedisKEY  = "string:integral_expire_deal_id"
const DealIdRedisTime = 60 * 60 * 24 * 7
const DealNum		  = 2
const GoroutineNum	  = 2
const RecordType	  = 1
const OrderType		  = 255
const TradePlatform   = 100
const DiscountType	  = 2
const IntegralPay	  = 1
const SourceType	  = 2
const Ad			  = 3
const BatchId		  = 1108
const RuleId		  = 441417

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
	Phone string `db:"phone" json:"phone"`
	GetIntegral int64 `db:"integral" json:"get_integral"`
	UseIntegral int64 `json:"use_integral"`
}

type userSum struct {
	Uid uint64 `json:"uid" db:"member_id"`
	IntegralSum float32 `json:"integral_sum" db:"integral_sum"`
}

func init() {
	redisObj = conf.RedisMaster()
	dbSlave  = conf.SqlSlaveDb()
	dbMaster = conf.SqlMasterDb()
	mx = new(sync.RWMutex)
	wg = new(sync.WaitGroup)
	rand.Seed(time.Now().UnixNano())
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
		var beginId uint64

		for {
			idRedis = getDealId()

			if idRedis >= idMysql {
				break
			}

			if idRedis <= 0 {
				idRedis = 1
				beginId = 1
			}

			atomic.StoreUint64(&beginId, idRedis)

			for i := 0; i < GoroutineNum; i ++ {
				wg.Add(1)
				go dealUserIntegral(beginId, beginTime, endTime)
				atomic.AddUint64(&beginId, DealNum)
			}

			setDealId(beginId)
			wg.Wait()
			time.Sleep(time.Second * 2)
			break
		}
	}
}



// 获取积分过期配置
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

// 是否处于过期清理时间
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

// 获取当前已经处理的ID
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

// 设置当前已经处理的ID
func setDealId(i uint64) {
	mx.Lock()
	defer mx.Unlock()

	redisObj.Set(DealIdRedisKEY, i, time.Second * DealIdRedisTime)
}

// 获取要处理的最大ID
func getBigDealId() uint64 {
	u := userIntegralExpire{}
	err := dbSlave.Get(&u, "select id from finance.user_expire_integral order by id desc limit 1")

	if err != nil {
		fmt.Println("获取积分过期表的ID出错：", err.Error())
		os.Exit(0)
	}

	return u.Id
}

// 处理用户积分过程
func dealUserIntegral(id uint64, expireBeginTime string, expireEndTime string) {
	defer wg.Done()

	beginId := id
	atomic.AddUint64(&id, DealNum)
	users := []userIntegralExpire{}
	err := dbSlave.Select(&users, "select id, uid, phone, integral from finance.user_expire_integral where id >= ? and id < ?",
		beginId, id)

	if err != nil {
		fmt.Println("在处理用户过期积分出错：", err.Error())
	} else {
		count := len(users)
		fmt.Println("清理的开始ID：", beginId, "，清理的用户数量：", count)

		var list = make(map[uint64]userIntegralExpire)

		for _, u := range users {
			fmt.Println("用户ID：", u.Uid, "；获取的积分：", u.GetIntegral)

			if u.GetIntegral > 0 {
				u.UseIntegral = calUserUsedIntegral(u.Uid, expireBeginTime, expireEndTime)
				list[u.Uid] = u
			}
		}

		if len(list) > 0 {
			for _, i := range list {
				i.cleanIntegral()
			}
		} else {
			fmt.Println("符合要求的用户没有，获取积分均为0")
		}
	}
}

// 计算用户使用的积分
func calUserUsedIntegral(uid uint64, expireBeginTime string, expireEndTime string) int64 {
	var useIntegral float32
	userIntegralInfo := userSum{}
	sql := "select member_id, sum(mabi_source) integral_sum from orders.integral_detail where member_id = ? and mabi_source < 0 " +
		"and create_time < ? and create_time > ? group by member_id"
	err := dbSlave.Get(&userIntegralInfo, sql, uid, expireEndTime, expireBeginTime)

	if err != nil {
		fmt.Println("获取用户积分出错：", err.Error())
		return 0
	} else {
		useIntegral = userIntegralInfo.IntegralSum
		fmt.Println("用户已使用的积分：", useIntegral)
		return int64(useIntegral)
	}
}

// 清除用户积分
func (u *userIntegralExpire)cleanIntegral() {
	cleanIntegral := u.GetIntegral + u.UseIntegral

	if cleanIntegral <= 0 {
		fmt.Println("用户：", u.Uid, " 积分不足以扣减：", u.GetIntegral, u.UseIntegral)
	} else {
		fmt.Println("开始进行用户：", u.Uid, "的积分扣减...")

		integralStr := ""
		dbSlave.QueryRow("select user_value from passport.user_meta where uid = ? and user_key = 'mabi'", u.Uid).Scan(&integralStr)
		integralInt, _ := strconv.Atoi(integralStr)
		fmt.Println("用户ID：", u.Uid, " 原有积分：", integralInt)
		recordNo := createNewRecordNo()
		sqlTime := getSqlTime()

		tx := dbMaster.MustBegin()
		tx.MustExec("update finance.user_expire_integral set pay_integral = ? where uid = ?", math.Abs(float64(u.UseIntegral)), u.Uid)

		tx.MustExec("insert into orders.discount_record(user_id, record_no, refund_amount, discount_amount, pay_amount, " +
			"total_amount, presented, order_id, record_type, spending_type, trade_platform, discount_type) values(?,?,?,?,?,?,?,?,?,?,?,?)",
				u.Uid, recordNo, 0, 0, 0, 0, cleanIntegral, "", RecordType, OrderType, TradePlatform, DiscountType)

		tx.MustExec("insert into finance.integral_discount_record(record_no, record_type, batch_id, rule_id, user_id, " +
			"user_phone, order_type, discount_integral, discount_amount, accounting_department, trade_platform) values(?,?,?,?,?,?,?,?,?,?,?)",
				recordNo, RecordType, BatchId, RuleId, u.Uid, u.Phone, OrderType, cleanIntegral, cleanIntegral, Ad, TradePlatform)

		tx.MustExec("update passport.user_meta set user_value = ? where uid = ? and user_key = 'mabi'", int64(integralInt) - cleanIntegral, u.Uid)

		tx.MustExec("insert into orders.integral_detail(member_id, surplus_mabi, mabi_source, type, create_time, update_time, " +
			"source_type, record_no, integral_code) values(?,?,?,?,?,?,?,?,?)",
				u.Uid, int64(integralInt) - cleanIntegral, -cleanIntegral, IntegralPay, sqlTime, sqlTime, SourceType, recordNo, "JFGQ")
		err := tx.Commit()

		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println("用户：", u.Uid, "的积分完毕，扣减了 ", cleanIntegral)
		}
	}
}

// 生成sql字段时间
func getSqlTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// 生成流水号
func createNewRecordNo() string {
	t := time.Now()
	year := strconv.Itoa(t.Year())

	recordNo := year[2:] + fmt.Sprintf("%02d", t.Month()) + fmt.Sprintf("%02d", t.Day()) +
		fmt.Sprintf("%02d", t.Hour()) + fmt.Sprintf("%02d", t.Minute()) + fmt.Sprintf("%02d", t.Second()) +
		fmt.Sprintf("%05d", rand.Intn(100000))
	return recordNo
}