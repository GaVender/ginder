package main

import (
	"time"
	"fmt"
	"ginder/conf"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/redis.v5"
	"strconv"
)

const MwUUIDRedis 				= "list:mw_sms_uuid"
const MwLastIdRedis 			= "string:mw_sms_id"
const WlUUIDRedis 				= "list:wl_sms_uuid"
const WlLastIdRedis 			= "string:wl_sms_id"
const MwMongoGetNum				= 80
const WlMongoGetNum				= 80
const SmsTypeMw 				= 2
const SmsTypeWl 				= 3
const SmsIdExpire 				= 60 * 60 * 24 * 30
const SmsWaitListChanLength 	= 100
const SmsSentListChanLength 	= 20000
const NotSend 					= 0
const Sent 						= 1
const MongoDatabase   			= "sms"
const MongoCollection 			= "batch_info_"
const RecoverSleepTime			= 3

var mwWaitSmsListChan = make(chan []SMS, SmsWaitListChanLength)
var wlWaitSmsListChan = make(chan []SMS, SmsWaitListChanLength)
var mwSentSmsListChan = make(chan SmsUpdate, SmsSentListChanLength)
var wlSentSmsListChan = make(chan SmsUpdate, SmsSentListChanLength)

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

func GetDataFromMongo(platform uint8) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("get data from mongo error and restart : ", err)
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
		fmt.Println("platform ", platform, " get sms last id error: ", err.Error())
	}

	for {
		uuid, err := getUUID(platform, redisObj)

		if err != nil {
			fmt.Println("platform ", platform, " get uuid error: ", err.Error())
			break
		}

		if "" == uuid {
			fmt.Println("platform ", platform, " uuid is empty, sms have sent over")
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
					fmt.Println("platform ", platform, " uuid: ", uuid, " has sent over")
					break
				} else {
					setSmsData(platform, &smsList)
				}

				setSmsLastSentId(platform, redisObj, beginId)
				fmt.Println("platform ", platform, " sms: ", smsList)
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
	if SmsTypeMw == platform {
		mwWaitSmsListChan <- *d
	} else if SmsTypeWl == platform {
		wlWaitSmsListChan <- *d
	} else {
		fmt.Println(ErrPlatform)
	}
}

func UpdateDataToMongo(platform uint8) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("update mongo error and restart : ", err)
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
			fmt.Println("platform: ", platform, " ", sms.ID, " update mongo fail：", err.Error())
			chanList <- sms
		} else {
			fmt.Println("platform: ", platform, " ", sms.ID, " update mongo success")
		}
	}
}