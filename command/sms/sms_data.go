package sms

import (
	"time"
	"fmt"
	"ginder/conf"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/redis.v5"
	"strconv"
	"ginder/framework/routinepool"
	"ginder/log/panellog"
)

const DateFormat				= "2006-01-02 15:04:05"
const NotSend 					= 0
const Sent 						= 1
const MongoDatabase   			= "sms"
const MongoCollection 			= "batch_info_"
const MwUUIDRedis 				= "list:mw_sms_uuid"
const MwLastIdRedis 			= "string:mw_sms_id"
const WlUUIDRedis 				= "list:wl_sms_uuid"
const WlLastIdRedis 			= "string:wl_sms_id"
const MwMongoGetNum				= 80
const WlMongoGetNum				= 80
const MwSendPoolSize			= 5
const MwSendPoolExpire			= 5
const MwUpdatePoolSize			= 5
const MwUpdatePoolExpire		= 5
const WlSendPoolSize			= 5
const WlSendPoolExpire			= 5
const WlUpdatePoolSize			= 5
const WlUpdatePoolExpire		= 5
const SmsTypeMw 				= 2
const SmsTypeWl 				= 3
const SmsIdExpire 				= 60 * 60 * 24 * 30
const SmsWaitListChanLength 	= 100
const SmsSentListChanLength 	= 20000
const SmsWaitListChanSleep		= 1
const RecoverSleepTime			= 3
const MonitorHeartBeatTime		= 5
const MonitorExpireTime			= 10


var mwWaitSmsListChan = make(chan []SMS, SmsWaitListChanLength)
var wlWaitSmsListChan = make(chan []SMS, SmsWaitListChanLength)
var mwSentSmsListChan = make(chan SmsUpdate, SmsSentListChanLength)
var wlSentSmsListChan = make(chan SmsUpdate, SmsSentListChanLength)

var getSmsProgramRunTime 	= make(map[uint8]int64)
var sendSmsProgramRunTime 	= make(map[uint8]int64)
var updateSmsProgramRunTime = make(map[uint8]int64)


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

type SmsUpdate struct {
	ID 		bson.ObjectId 	`json:"id" bson:"_id"`
	MsgId 	string			`json:"msg_id" bson:"msg_id"`
}


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

	mongo := conf.MongoSession()
	defer mongo.Close()

	redisObj := conf.RedisMaster()
	defer redisObj.Close()

	ts  := time.Now().Format("200601")
	con := mongo.DB(MongoDatabase).C(MongoCollection + ts)
	sms := SMS{}

	beginId, err := getSmsLastSentId(platform, redisObj)

	if err != nil {
		panellog.SmsPanelLog.Log("getSmsError", "platform ", platform, " get sms last id error: ", err.Error())
	}

	for {
		getSmsProgramRunTime[platform] = time.Now().Unix()

		uuid, err := getUUID(platform, redisObj)

		if err != nil {
			panellog.SmsPanelLog.Log("getSmsError", "platform ", platform, " get uuid error: ", err.Error())
			break
		}

		if "" == uuid {
			panellog.SmsPanelLog.Log("getSms", "platform ", platform, " uuid is empty, sms have sent over")
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
		updateSmsProgramRunTime[platform] = time.Now().Unix()

		pool.Submit(func() error {
			UpdateDataToMongo(platform)
			return nil
		})
	}
}

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
		mongo := conf.MongoSession()
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

		for k, v := range getSmsProgramRunTime {
			if (ts - v) > MonitorExpireTime {
				panellog.SmsPanelLog.Log("monitor", "program of platform ", k, " getting sms from mongo is stop")
			}
		}

		for k, v := range sendSmsProgramRunTime {
			if (ts - v) > MonitorExpireTime {
				panellog.SmsPanelLog.Log("monitor","program of platform ", k, " sending sms is blocking")
			}
		}

		for k, v := range updateSmsProgramRunTime {
			if (ts - v) > MonitorExpireTime {
				panellog.SmsPanelLog.Log("monitor","program of platform ", k, " updating sms to mongo is blocking")
			}
		}
	}
}