package conf

import (
	"fmt"
	"os"
	"strings"
	"time"

	log4 "github.com/GaVender/ginder/log/log4go"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"gopkg.in/mgo.v2"
	"gopkg.in/redis.v5"
)

const (
	Master = "master"
	Slave  = "slave"

	Log4 = "log4"
)

var (
	masterRedisClient *redis.Client
	slaveRedisClient  *redis.Client
	masterMysqlClient *sqlx.DB
	slaveMysqlClient  *sqlx.DB
	mongoClient 	  *mgo.Session
)

var (
	mysqlMasterHost 		string
	mysqlSlaveHost  		string
	mysqlPort 	      		string
	mysqlUsername    		string
	mysqlPwd	  			string
	mysqlDB		  			string
	redisMasterHost 		string
	redisSlaveHost  		string
	redisPort		  		string
	redisPwd	  			string
	redisDb		  			int
	redisPoolSize   		int
	mongoHost 				[]string
	mongoUser				string
	mongoPwd				string
	mongoDB					string
	mongoTimeout			int
	mongoPoolLimit			int
	loggerForLogic 			LogProvider
	loggerForError 			LogProvider
)

type LogProvider interface {
	Close()
	LogInfo(head, content string)
	LogWarn(head, content string)
	LogError(head, content string)
}

func init() {
	/*
		参数初始化
	*/
	mysqlMasterHost	  	= strings.TrimSpace(os.Getenv("MYSQL_MASTER_HOST"))
	mysqlSlaveHost 	  	= strings.TrimSpace(os.Getenv("MYSQL_MASTER_HOST"))
	mysqlPort 		  	= strings.TrimSpace(os.Getenv("MYSQL_PORT"))
	mysqlUsername 	  	= strings.TrimSpace(os.Getenv("MYSQL_USERNAME"))
	mysqlPwd 		  	= strings.TrimSpace(os.Getenv("MYSQL_PASSWORD"))
	mysqlDB 		  	= strings.TrimSpace(os.Getenv("MYSQL_DB"))
	redisMasterHost   	= strings.TrimSpace(os.Getenv("REDIS_HOST"))
	redisSlaveHost 	  	= strings.TrimSpace(os.Getenv("REDIS_HOST"))
	redisPort 		  	= strings.TrimSpace(os.Getenv("REDIS_PORT"))
	redisPwd 		  	= strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	redisDb		  		= 0
	redisPoolSize	  	= 20
	mongoHost 		  	= []string{strings.TrimSpace(os.Getenv("MONGO_HOST"))}
	mongoUser 		  	= strings.TrimSpace(os.Getenv("MONGO_USER"))
	mongoPwd 		  	= strings.TrimSpace(os.Getenv("MONGO_PASSWORD"))
	mongoDB				= strings.TrimSpace(os.Getenv("MONGO_DB"))
	mongoTimeout	   	= 1
	mongoPoolLimit		= 20

	/*
		日志配置，一个是系统级别错误，一个是业务逻辑错误
	*/
	getLogger(Log4)

	/*
		启动mysql、redis、mongo配置
	*/
	masterMysqlClient = getMysql(Master)
	slaveMysqlClient  = getMysql(Slave)
	masterRedisClient = getRedis(Master)
	slaveRedisClient  = getRedis(Slave)
	mongoSessionFactory()
}

func getLogger (name string) {
	switch name {
	case "log4":
		loggerForError = log4.GetErrorLogger()
		loggerForLogic = log4.GetLogicLogger()
	default:
		loggerForError = log4.GetErrorLogger()
		loggerForLogic = log4.GetLogicLogger()
	}
}

func GetErrorLogger() LogProvider {
	return loggerForError
}

func GetLogicLogger() LogProvider {
	return loggerForLogic
}

func mysqlFactory(host string) *sqlx.DB {
	if Master == host {
		host = mysqlMasterHost
	} else {
		host = mysqlSlaveHost
	}

	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", mysqlUsername, mysqlPwd,
		host, mysqlPort, mysqlDB)
	db, err := sqlx.Connect("mysql", dns)

	if err != nil {
		loggerForError.LogError("mysql", fmt.Sprintf("%s connect error: %s", dns, err.Error()))
		panic("mysql connect error: " + err.Error())
	} else {
		loggerForError.LogInfo("mysql", host + " mysql start success ...")
	}

	return db
}

func getMysql(host string) *sqlx.DB {
	return mysqlFactory(host)
}

func GetMasterMysql() *sqlx.DB {
	return masterMysqlClient
}

func CloseMasterMysql() {
	if err := masterMysqlClient.Close(); err != nil {
		loggerForError.LogError("mysql", fmt.Sprintf("master close error: %s", err.Error()))
		panic("master mysql close error: " + err.Error())
	}
}

func GetSlaveMysql() *sqlx.DB {
	return slaveMysqlClient
}

func CloseSlaveMysql() {
	if err := slaveMysqlClient.Close(); err != nil {
		loggerForError.LogError("mysql", fmt.Sprintf("slave close error: %s", err.Error()))
		panic("slave mysql close error: " + err.Error())
	}
}

func redisFactory(host string) *redis.Client {
	if Master == host {
		host = redisMasterHost
	} else {
		host = redisSlaveHost
	}

	r := redis.NewClient(&redis.Options{
		Addr:        fmt.Sprintf("%s:%s", host, redisPort),
		Password:    redisPwd,
		DB:          redisDb,
		PoolSize:    redisPoolSize,
	})

	if _, err := r.Ping().Result(); err != nil {
		loggerForError.LogError("redis", fmt.Sprintf("connect error: %s", err.Error()))
		panic("redis start error: " + err.Error())
	} else {
		fmt.Println(host + " redis start success ...")
	}

	return r
}

func getRedis(name string) *redis.Client {
	return redisFactory(name)
}

func GetMasterRedis() *redis.Client {
	return masterRedisClient
}

func CloseMasterRedis() {
	if err := masterRedisClient.Close(); err != nil {
		loggerForError.LogError("redis", fmt.Sprintf("master close error: %s", err.Error()))
		panic("master redis close error: " + err.Error())
	}
}

func GetSlaveRedis() *redis.Client {
	return slaveRedisClient
}

func CloseSlaveRedis() {
	if err := slaveRedisClient.Close(); err != nil {
		loggerForError.LogError("redis", fmt.Sprintf("slave close error: %s", err.Error()))
		panic("slave redis close error: " + err.Error())
	}
}

func mongoSessionFactory() {
	var err error

	dialInfo := &mgo.DialInfo{
		Addrs:     mongoHost,
		Username:  mongoUser,
		Password:  mongoPwd,
		Database:  mongoDB,
		Timeout:   time.Second * time.Duration(mongoTimeout),
		Direct:    false,
		PoolLimit: mongoPoolLimit,
	}

	mongoClient, err = mgo.DialWithInfo(dialInfo)

	if err != nil {
		loggerForError.LogError("mongo", fmt.Sprintf("connect error: %s", err.Error()))
		panic("mongo connect error: " + err.Error())
	} else {
		fmt.Println("mongo start success ...")
		mongoClient.SetMode(mgo.Eventual, true)
	}
}

func CloseMongoSession() {
	mongoClient.Close()
}

func GetMongoSession() *mgo.Session {
	if mongoClient == nil {
		mongoSessionFactory()
	}

	return mongoClient.Copy()
}