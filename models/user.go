package models

import (
	"ginder/conf"
)

type User struct {
	Id 			   int64 `db:"id" json:"id"`
	Username 	   string `db:"username" json:"username"`
	Password 	   string `db:"password" json:"password"`
	CreateTime 	   string `db:"create_time" json:"create_time"`
	LastUpdateTime string `db:"last_update_time" json:"last_update_time"`
	IsDeleted 	   string `db:"is_deleted" json:"is_deleted"`
	InviteCode 	   string `db:"invite_code" json:"invite_code"`
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
		conf.LoggerLogic().Error("mysql select user exists error: %s", err.Error())
	} else {
		if user.Id != 0 {
			flag = 1
		} else {
			flag = 0
		}
	}

	return flag
}