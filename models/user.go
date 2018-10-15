package models

import (
	"ginder/conf"
	"crypto/md5"
	"fmt"
	"strings"
	"database/sql"
	"time"
)

const PWD_ENCRYPT_PREFIX = "linge"

type User struct {
	Id 			   int64  `db:"id" json:"id"`
	Username 	   string `db:"username" json:"username"`
	Password 	   string `db:"password" json:"password"`
	CreateTime 	   string `db:"create_time" json:"create_time"`
	LastUpdateTime string `db:"last_update_time" json:"last_update_time"`
	IsDeleted 	   string `db:"is_deleted" json:"is_deleted"`
	InviteCode 	   string `db:"invite_code" json:"invite_code"`
}

type UserInfoRedis struct {
	Token 	 string `json:"token"`
	Uid 	 int64  `json:"uid"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserLogin struct {
	Username string
	Password string
}



func (u *UserLogin) IsExists() int8 {
	db := conf.SqlSlaveDb()
	defer db.Close()

	var flag int8 = -1
	var user User
	err := db.Get(&user, "select id from passport.user where username = ?", u.Username)

	if err != nil {
		if strings.Index(err.Error(), "no rows in result set") >= 0 {
			flag = 0
		} else {
			conf.LoggerLogic().Error("mysql select user exists error: %s", err.Error())
		}
	} else {
		if user.Id != 0 {
			flag = 1
		} else {
			flag = 0
		}
	}

	return flag
}

func (u *UserLogin) Register() (sql.Result, error) {
	db := conf.SqlMasterDb()
	defer db.Close()

	query := "insert into passport.user(username, password) values(?, ?);"
	return db.Exec(query, u.Username, u.EncryptPassword())
}

func (u *UserLogin) UserInfo() *User {
	user := User{}

	db := conf.SqlSlaveDb()
	defer db.Close()
	db.Get(&user, "select id, username, password from passport.user where username = ?", u.Username)
	return &user
}

func (u *UserLogin) EncryptPassword() string {
	if u.Password != "" {
		pwd := fmt.Sprintf("%s_%s", PWD_ENCRYPT_PREFIX, u.Password)
		return fmt.Sprintf("%x", md5.Sum([]byte(pwd)))
	} else {
		return ""
	}
}

func (u *UserLogin) CreateUserToken() string {
	if u.Username != "" {
		token := fmt.Sprintf("%s_%s_%s", PWD_ENCRYPT_PREFIX, u.Username, time.Now().Unix())
		return fmt.Sprintf("%x", md5.Sum([]byte(token)))
	} else {
		return ""
	}
}