package main

import (
	"net/http"
	"strings"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"time"
	"crypto/md5"
	"errors"
	"gopkg.in/mgo.v2/bson"
	"github.com/henrylee2cn/mahonia"
	"net/url"
	"os"
)

var MwSmsUrl = os.Getenv("MW_SMS_URL")
var MwSmsSp  = os.Getenv("MW_SMS_SP")
var MwSmsPwd = os.Getenv("MW_SMS_PWD")

type Sms struct {
	Id 		bson.ObjectId 	`json:"id"`
	Phone   string 			`json:"phone"`
	Content string 			`json:"content"`
	UUID    string 			`json:"uuid"`
}

type BatchSms []Sms

type sender interface {
	send() error
	sendData(*[]SMS) error
	dealData(*BatchSms) string
}

type MwSmsStruct struct {
	Userid 		string 		`json:"userid"`
	Pwd 		string 		`json:"pwd"`
	Timestamp 	string 		`json:"timestamp"`
	Multimt 	[]MwMultimt `json:"multimt"`
}

type MwMultimt struct {
	Mobile 	string `json:"mobile"`
	Content string `json:"content"`
	Svrtype string `json:"svrtype"`
	Exno 	string `json:"exno"`
	Custid 	string `json:"custid"`
	Exdata 	string `json:"exdata"`
}

type MwResp struct {
	Result int32 	`json:"result"`
	MsgId  int64 	`json:"msgid"`
	CustId string 	`json:"custid"`
}

type MwSms struct {}

type WlSms struct {}

func SendSms(platform uint8) error {
	var s sender

	switch platform {
		case 2:
			s = &MwSms{}
			break
		case 3:
			s = &WlSms{}
			break
		default:
			return ErrPlatform
	}

	s.send()
	return nil
}

func (s *MwSms) send() error {
	select {
	case smsList := <- mwWaitSmsListChan:
		if len(smsList) > 0 {
			err := s.sendData(&smsList)

			if err != nil {
				fmt.Println("梦网发送失败：", err.Error())
			}
		}
	default:
		fmt.Println("sms wait chan no data")
		time.Sleep(time.Second)
	}

	return nil
}

func (s *MwSms) sendData(smsList *[]SMS) error {
	b := BatchSms{}
	enc := mahonia.NewEncoder("gbk")

	for _, v := range *smsList {
		b = append(b, Sms{Id: v.ID, Phone: v.Phone, Content: url.QueryEscape(enc.ConvertString(v.Content)), UUID: v.UUID})
	}

	resp, err := http.Post(MwSmsUrl, "application/json", strings.NewReader(s.dealData(&b)))

	if err != nil{
		return errors.New("梦网接口出错：" + err.Error())
	} else {
		r := MwResp{}
		respBody, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return errors.New("梦网获取数据出错：" + err.Error())
		}

		if err := json.Unmarshal(respBody, &r); err != nil {
			return errors.New("梦网返回数据解析出错：" + err.Error())
		} else {
			fmt.Println("梦网返回数据：", r)

			if r.Result != 0 {
				mwWaitSmsListChan <- *smsList
				return errors.New(fmt.Sprintf("梦网发送出错：%d", r.Result))
			} else {
				s.saveData(&b, r.MsgId)
			}
		}
	}

	return nil
}

func (s *MwSms) dealData(b *BatchSms) string {
	t   := time.Now()
	ts 	:= t.Format("0102150405")
	pwd := MwSmsSp + "00000000" + MwSmsPwd + ts
	pwd  = fmt.Sprintf("%x", md5.Sum([]byte(pwd)))
	param := MwSmsStruct{Userid: MwSmsSp, Pwd: pwd, Timestamp: ts}
	
	var ml []MwMultimt

	for _, v := range *b {
		m := MwMultimt{Mobile: v.Phone, Content: v.Content, Custid: v.UUID}
		ml = append(ml, m)
	}

	param.Multimt = ml
	data, _ := json.Marshal(param)
	return string(data)
}

func (s *MwSms) saveData(b *BatchSms, msgId int64) error {
	if len(*b) > 0 {
		for _, v := range *b {
			sp := SmsUpdate{ID: v.Id, MsgId: fmt.Sprintf("%d", msgId)}

			select {
			case mwSentSmsListChan <- sp:
				fmt.Println("短信发完，放入send chan：", sp.ID)
			case <- time.After(time.Microsecond * 50):
				fmt.Println("短信发完，放入send chan 超时，已丢弃：", sp.ID)
			}
		}
	}

	return nil
}

func (s *WlSms) send() error {
	return nil
}

func (s *WlSms) sendData(smsList *[]SMS) error {
	return nil
}

func (s *WlSms) dealData(b *BatchSms) string {
	return ""
}

func (s *WlSms) saveData(b *BatchSms) error {
	return nil
}