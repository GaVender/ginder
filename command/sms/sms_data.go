package sms

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/GaVender/ginder/conf"
	"github.com/GaVender/ginder/framework/routinepool"
	"github.com/GaVender/ginder/log/panellog"

	"gopkg.in/mgo.v2/bson"
	"gopkg.in/redis.v5"
)

const DateFormat				= "2006-01-02 15:04:05"
const NotSend 					= 0
const Sent 						= 1
const MongoDatabase   			= "sms"										// 存储短信的mongo database
const MongoCollection 			= "batch_info_"								// 存储批量短信的mongo collection
const MwUUIDRedis 				= "list:mw_sms_uuid"						// 梦网存储的批次uuid的redis key
const MwLastIdRedis 			= "string:mw_sms_id"						// 梦网存储最后发送的短信的object_id redis key
const WlUUIDRedis 				= "list:wl_sms_uuid"						// 未来存储的批次uuid的redis key
const WlLastIdRedis 			= "string:wl_sms_id"						// 未来存储最后发送的短信的object_id redis key
const MwMongoGetNum				= 80										// 梦网一次性从mongo获取的要发送短信的数量
const WlMongoGetNum				= 80										// 未来一次性从mongo获取的要发送短信的数量
const MwSendPoolSize			= 5											// 梦网发送协程池的容量
const MwSendPoolExpire			= 5											// 梦网发送协程池的监控频率
const MwUpdatePoolSize			= 5											// 梦网更新协程池的容量
const MwUpdatePoolExpire		= 5											// 梦网更新协程池的监控频率
const WlSendPoolSize			= 5											// 未来发送协程池的容量
const WlSendPoolExpire			= 5											// 未来发送协程池的监控频率
const WlUpdatePoolSize			= 5											// 未来更新协程池的容量
const WlUpdatePoolExpire		= 5											// 未来更新协程池的监控频率
const SmsTypeMw 				= 2
const SmsTypeWl 				= 3
const SmsIdExpire 				= 60 * 60 * 24 * 30							// 存储短信object_id的过期时间
const SmsWaitListChanLength 	= 100										// 存储即将发送短信的wait队列长度
const SmsSentListChanLength 	= 20000										// 存储已发送短信的sent队列长度
const SmsWaitListChanSleep		= 1											// wait队列放满时的休息时间，之后重新存储
const RecoverSleepTime			= 3											// 程序panic将重启时的休息时间
const MonitorHeartBeatTime		= 5											// 监控频率
const MonitorExpireTime			= 10										// 监控的缓冲时间


var mwWaitSmsListChan = make(chan []SMS, SmsWaitListChanLength)				// 梦网即将发送短信的等待队列
var wlWaitSmsListChan = make(chan []SMS, SmsWaitListChanLength)				// 未来即将发送短信的等待队列
var mwSentSmsListChan = make(chan SmsUpdate, SmsSentListChanLength)			// 梦网已发送短信的更新队列
var wlSentSmsListChan = make(chan SmsUpdate, SmsSentListChanLength)			// 未来已发送短信的更新队列

var getSmsProgramLock		= sync.Mutex{}									// 获取短信程序的锁
var getSmsProgramRunTime 	= make(map[uint8]int64)							// 获取短信程序的运行时间
var sendSmsProgramLock		= sync.Mutex{}									// 发送短信程序的锁
var sendSmsProgramRunTime 	= make(map[uint8]int64)							// 发送短信程序的运行时间
var updateSmsProgramLock	= sync.Mutex{}									// 更新短信程序的锁
var updateSmsProgramRunTime = make(map[uint8]int64)							// 更新短信程序的运行时间


// 短信存储在mongo中的格式
type SMS struct {
	ToPlatform 			int8 			`json:"to_platform" bson:"to_platform"`
	ToOperator 			int8 			`json:"to_operator" bson:"to_operator"`
	ToUser 				int8 			`json:"to_user" bson:"to_user"`
	PlatformType 		uint8 			`json:"platform_type" bson:"platform_type"`
	Num 				uint8 			`json:"num" bson:"num"`
	CreateTime 			uint32 			`json:"create_time" bson:"create_time"`
	LastUpdateTime 		uint32 			`json:"last_update_time" bson:"last_update_time"`
	ID 					bson.ObjectId 	`json:"id" bson:"_id"`
	BatchId				string 			`json:"batch_id" bson:"batch_id"`
	Phone 				string			`json:"phone" bson:"phone"`
	Content 			string			`json:"content" bson:"content"`
	UUID 				string			`json:"uuid" bson:"uuid"`
	MsgId 				string			`json:"msg_id" bson:"msg_id"`
	ErrMsg 				string			`json:"err_msg" bson:"err_msg"`
}

// 短信存储在更新chan中的格式
type SmsUpdate struct {
	ID 		bson.ObjectId 	`json:"id" bson:"_id"`
	MsgId 	string			`json:"msg_id" bson:"msg_id"`
}


// 短信的整个发送运行过程
func SendProcedure(platform uint8) {
	defer func() {
		if err := recover(); err != nil {
			panellog.SmsPanelLog.Log("panic", "Start send sms whole procedure error and restart: ", err)
			time.Sleep(time.Second * RecoverSleepTime)
			SendProcedure(platform)
		}
	}()

	if SmsTypeMw == platform || SmsTypeWl == platform {
		go GetDataFromMongo(platform)
		go CreateSendPool(platform)
		go CreateUpdatePool(platform)
	} else {
		panic("error sms platform: " + strconv.Itoa(int(platform)))
	}
}

// 从mongo获取即将发送的短信，格式化后存储在chan中
func GetDataFromMongo(platform uint8) {
	defer func() {
		if err := recover(); err != nil {
			panellog.SmsPanelLog.Log("panic", "get data from mongo error and restart: ", err)
			time.Sleep(time.Second * RecoverSleepTime)
			GetDataFromMongo(platform)
		}
	}()

	var batchGetNumFromMongo uint16

	if SmsTypeMw == platform {
		batchGetNumFromMongo = MwMongoGetNum
	} else if SmsTypeWl == platform {
		batchGetNumFromMongo = WlMongoGetNum
	} else {
		panic("error sms platform: " + strconv.Itoa(int(platform)))
	}

	mongo := conf.GetMongoSession()
	defer mongo.Close()

	redisObj := conf.GetMasterRedis()

	ts  := time.Now().Format("200601")
	con := mongo.DB(MongoDatabase).C(MongoCollection + ts)
	sms := SMS{}

	beginId, err := getSmsLastSentId(platform, redisObj)

	if err != nil {
		panellog.SmsPanelLog.Log("getSmsError", "platform ", platform, " get sms last id error: ", err.Error())
	}

	for {
		getSmsProgramLock.Lock()
		getSmsProgramRunTime[platform] = time.Now().Unix()
		getSmsProgramLock.Unlock()

		uuid, err := getUUID(platform, redisObj)

		if err != nil {
			panellog.SmsPanelLog.Log("getSmsError", "platform ", platform, " get uuid error: ", err.Error())
			break
		}

		if "" == uuid {
			panellog.SmsPanelLog.Log("getSmsError", "platform ", platform, " uuid is empty, sms have sent over")
		} else {
			for {
				smsList := []SMS{}

				i := con.Find(bson.M{
					"_id":           bson.M{"$gt": beginId},
					"uuid":          uuid,
					"platform_type": platform,
					"to_platform":   NotSend,
				}).Sort("_id").Limit(int(batchGetNumFromMongo)).Iter()

				for i.Next(&sms) {
					smsList = append(smsList, sms)
					beginId = sms.ID
				}

				if len(smsList) <= 0 {
					panellog.SmsPanelLog.Log("getSms", "platform ", platform, " uuid: ", uuid, " has sent over")
					break
				} else {
					setSmsData(platform, &smsList)
				}

				setSmsLastSentId(platform, redisObj, beginId)
				panellog.SmsPanelLog.Log("getSms", "platform ", platform, " sms: ", smsList)
			}
		}

		time.Sleep(time.Second * RecoverSleepTime)
	}
}

// 从redis获取每一批短信的uuid，uuid在php的rpc接口创建
func getUUID(platform uint8, redisObj *redis.Client) (string, error) {
	var redisName string

	if SmsTypeMw == platform {
		redisName = MwUUIDRedis
	} else if SmsTypeWl == platform {
		redisName = WlUUIDRedis
	} else {
		return "", ErrPlatform
	}

	uuid := redisObj.LPop(redisName).Val()
	return uuid, nil
}

// 获取最近发送的短信object_id
func getSmsLastSentId(platform uint8, redisObj *redis.Client) (bson.ObjectId, error) {
	var redisName string

	if SmsTypeMw == platform {
		redisName = MwLastIdRedis
	} else if SmsTypeWl == platform {
		redisName = WlLastIdRedis
	} else {
		return "", ErrPlatform
	}

	id := redisObj.Get(redisName).Val()

	if "" == id {
		return bson.ObjectIdHex("5c000000c0570561793331c0"), nil
	} else {
		return bson.ObjectIdHex(id), nil
	}
}

// 保存最近发送的短信object_id
func setSmsLastSentId(platform uint8, redisObj *redis.Client, id bson.ObjectId) error {
	var redisName string

	if SmsTypeMw == platform {
		redisName = MwLastIdRedis
	} else if SmsTypeWl == platform {
		redisName = WlLastIdRedis
	} else {
		return ErrPlatform
	}

	redisObj.Set(redisName, id.Hex(), time.Second * SmsIdExpire)
	return nil
}

// 保存即将发送的短信到chan
func setSmsData(platform uint8, d *[]SMS) {
	var listChan chan []SMS

	if SmsTypeMw == platform {
		listChan = mwWaitSmsListChan
	} else if SmsTypeWl == platform {
		listChan = wlWaitSmsListChan
	} else {
		fmt.Println(ErrPlatform)
		return
	}

	flag := false

	for true {
		select {
		case listChan <- *d:
			flag = true
			break
		default:
			panellog.SmsPanelLog.Log("getSms", "platform ", platform, " wait chan is full, can not put in data")
			time.Sleep(time.Second * SmsWaitListChanSleep)
		}

		if flag {
			break
		}
	}
}

// 创建更新短信的协程池
func CreateUpdatePool(platform uint8) {
	defer func() {
		if err := recover(); err != nil {
			panellog.SmsPanelLog.Log("panic", "update sms pool error and restart : ", err)
			time.Sleep(time.Second * RecoverSleepTime)
			CreateUpdatePool(platform)
		}
	}()

	var pool *routinepool.Pool
	var err error

	if SmsTypeMw == platform {
		pool, err = routinepool.NewPool(MwUpdatePoolSize, MwUpdatePoolExpire)
	} else if SmsTypeWl == platform {
		pool, err = routinepool.NewPool(WlUpdatePoolSize, WlUpdatePoolExpire)
	} else {
		panic("error sms platform: " + strconv.Itoa(int(platform)))
	}

	if err != nil {
		panic("platform " + strconv.Itoa(int(platform)) + " create pool error: " + err.Error())
	} else {
		panellog.SmsPanelLog.Log("updateSms", "create platform ", platform, " update pool success...")
	}

	for true {
		updateSmsProgramLock.Lock()
		updateSmsProgramRunTime[platform] = time.Now().Unix()
		updateSmsProgramLock.Unlock()

		pool.Submit(func() error {
			UpdateDataToMongo(platform)
			return nil
		})
	}
}

// 更新发送后mongo中的短信状态
func UpdateDataToMongo(platform uint8) {
	defer func() {
		if err := recover(); err != nil {
			panellog.SmsPanelLog.Log("panic", "update mongo error and restart : ", err)
			time.Sleep(time.Second * RecoverSleepTime)
			UpdateDataToMongo(platform)
		}
	}()

	var chanList chan SmsUpdate

	if SmsTypeMw == platform {
		chanList = mwSentSmsListChan
	} else if SmsTypeWl == platform {
		chanList = wlSentSmsListChan
	} else {
		panic("error platform: " + strconv.Itoa(int(platform)))
	}

	sms := <- chanList

	if sms.ID != "" {
		mongo := conf.GetMongoSession()
		defer mongo.Close()

		ts  := time.Now().Format("200601")
		con := mongo.DB(MongoDatabase).C(MongoCollection + ts)
		err := con.UpdateId(sms.ID, bson.M{
			"$set": bson.M{
				"to_platform": Sent, "msg_id": sms.MsgId, "last_update_time": int32(time.Now().Unix()),
			},
		})

		if err != nil {
			panellog.SmsPanelLog.Log("updateSmsError", "platform: ", platform, " ", sms.ID, " update mongo fail：", err.Error())
			chanList <- sms
		} else {
			panellog.SmsPanelLog.Log("updateSms", "platform: ", platform, " ", sms.ID, " update mongo success")
		}
	}
}

// 监控整个流程
func Monitor() {
	defer func() {
		if err := recover(); err != nil {
			panellog.SmsPanelLog.Log("panic", "monitor program error and restart: ", err)
			time.Sleep(time.Second * RecoverSleepTime)
			Monitor()
		}
	}()

	heartBeat := time.NewTicker(time.Second * MonitorHeartBeatTime)
	defer heartBeat.Stop()

	for range heartBeat.C {
		ts := time.Now().Unix()
		panellog.SmsPanelLog.Log("monitor", "monitor time")

		getSmsProgramLock.Lock()
		for k, v := range getSmsProgramRunTime {
			if (ts - v) > MonitorExpireTime {
				panellog.SmsPanelLog.Log("monitor", "program of platform ", k, " getting sms from mongo is stop")
			}
		}
		getSmsProgramLock.Unlock()

		sendSmsProgramLock.Lock()
		for k, v := range sendSmsProgramRunTime {
			if (ts - v) > MonitorExpireTime {
				panellog.SmsPanelLog.Log("monitor","program of platform ", k, " sending sms is blocking")
			}
		}
		sendSmsProgramLock.Unlock()

		updateSmsProgramLock.Lock()
		for k, v := range updateSmsProgramRunTime {
			if (ts - v) > MonitorExpireTime {
				panellog.SmsPanelLog.Log("monitor","program of platform ", k, " updating sms to mongo is blocking")
			}
		}
		updateSmsProgramLock.Unlock()
	}
}