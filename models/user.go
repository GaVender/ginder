package models

type User struct {
	Id 			   int `db:"id" json:"id"`
	Username 	   string `db:"username" json:"username"`
	Password 	   string `db:"password" json:"password"`
	CreateTime 	   string `db:"create_time" json:"create_time"`
	LastUpdateTime string `db:"last_update_time" json:"last_update_time"`
	IsDeleted 	   string `db:"is_deleted" json:"is_deleted"`
	InviteCode 	   string `db:"invite_code" json:"invite_code"`
}