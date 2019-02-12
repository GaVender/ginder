package sms

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
	"strconv"
	"ginder/framework/routinepool"
	"ginder/log/panellog"
)

var MwSmsUrl = os.Getenv("MW_SMS_URL")
var MwSmsSp  = os.Getenv("MW_SMS_SP")
var MwSmsPwd = os.Getenv("MW_SMS_PWD")

var WlSmsUrl = os.Getenv("WL_SMS_URL")
var WlSmsSp  = os.Getenv("WL_SMS_SP")
var WlSmsPwd = os.Getenv("WL_SMS_PWD")
var WlSmsSrc = os.Getenv("WL_SMS_SRCPHONE")


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

type WlSmsStruct struct {
	Uid 	 string `json:"uid"`
	Sign 	 string `json:"sign"`
	Srcphone string `json:"srcphone"`
	Msg 	 string `json:"msg"`
}

type WlSmsMsg struct {
	Phone 	string `json:"phone"`
	Context string `json:"context"`
}

type WlResp string

type MwSms struct {}

type WlSms struct {}


func CreateSendPool(platform uint8) {
	defer func() {
		if err := recover(); err != nil {
			panellog.SmsPanelLog.Log("panic", "create sms pool error and restart : ", err)
			time.Sleep(time.Second * RecoverSleepTime)
			CreateSendPool(platform)
		}
	}()

	var pool *routinepool.Pool
	var err error

	if SmsTypeMw == platform {
		pool, err = routinepool.NewPool(MwSendPoolSize, MwSendPoolExpire)
	} else if SmsTypeWl == platform {
		pool, err = routinepool.NewPool(WlSendPoolSize, WlSendPoolExpire)
	} else {
		panic("error sms platform: " + strconv.Itoa(int(platform)))
	}

	if err != nil {
		panic("platform " + strconv.Itoa(int(platform)) + " create pool error: " + err.Error())
	} else {
		panellog.SmsPanelLog.Log("sendSms", "create platform ", platform, " send pool success...")
	}

	for true {
		sendSmsProgramRunTime[platform] = time.Now().Unix()

		pool.Submit(func() error {
			SendSms(platform)
			return nil
		})
	}
}

func SendSms(platform uint8) {
	defer func() {
		if err := recover(); err != nil {
			panellog.SmsPanelLog.Log("panic", "send sms error and restart : ", err)
			time.Sleep(time.Second * RecoverSleepTime)
			SendSms(platform)
		}
	}()

	var s sender

	switch platform {
		case SmsTypeMw:
			s = &MwSms{}
			break
		case SmsTypeWl:
			s = &WlSms{}
			break
		default:
			panic("error platform: " + strconv.Itoa(int(platform)))
	}

	s.send()
}

func (s *MwSms) send() error {
	smsList := <- mwWaitSmsListChan

	if len(smsList) > 0 {
		err := s.sendData(&smsList)

		if err != nil {
			panellog.SmsPanelLog.Log("sendSmsError", "mw sent error：", err.Error())
		}
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
		return errors.New("mw interface error：" + err.Error())
	} else {
		r := MwResp{}
		respBody, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return errors.New("mw response error：" + err.Error())
		}

		if err := json.Unmarshal(respBody, &r); err != nil {
			return errors.New("mw response analysis error：" + err.Error())
		} else {
			fmt.Println("mw response：", r)

			if r.Result != 0 {
				mwWaitSmsListChan <- *smsList
				return errors.New(fmt.Sprintf("mw sent error：%d", r.Result))
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
				panellog.SmsPanelLog.Log("sendSms", "mw sms sent success and put in sent chan：", sp.ID)
			case <- time.After(time.Microsecond * 50):
				panellog.SmsPanelLog.Log("sendSms", "mw sms sent success but put in sent chan overtime：", sp.ID)
			}
		}
	}

	return nil
}

func (s *WlSms) send() error {
	smsList := <- wlWaitSmsListChan

	if len(smsList) > 0 {
		err := s.sendData(&smsList)

		if err != nil {
			panellog.SmsPanelLog.Log("sendSmsError", "wl sent error：", err.Error())
		}
	}

	return nil
}

func (s *WlSms) sendData(smsList *[]SMS) error {
	b := BatchSms{}

	for _, v := range *smsList {
		b = append(b, Sms{Id: v.ID, Phone: v.Phone, Content: v.Content, UUID: v.UUID})
	}

	resp, err := http.Post(WlSmsUrl, "application/json", strings.NewReader(s.dealData(&b)))

	if err != nil{
		return errors.New("wl interface error：" + err.Error())
	} else {
		//var r WlResp
		respBody, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return errors.New("wl response error：" + err.Error())
		}
fmt.Println(url.QueryUnescape(string(respBody)))
return nil
		/*if err := json.Unmarshal(respBody, &r); err != nil {
			return errors.New("wl response analysis error：" + err.Error())
		} else {
			fmt.Println("wl response：", r)

			if r.ID != 0 {
				//mwWaitSmsListChan <- *smsList
				return errors.New(fmt.Sprintf("wl sent error：%d", r))
			} else {
				//s.saveData(&b, r.MsgId)
			}
		}*/
	}

	return nil
}

func (s *WlSms) dealData(b *BatchSms) string {
	var msg []WlSmsMsg

	for _, v := range *b {
		w := WlSmsMsg{Phone: v.Phone, Context: v.Content}
		msg = append(msg, w)
	}

	msgByte, _ := json.Marshal(msg)
	msgStr := string(msgByte)
	msgStr = strings.Replace(msgStr, "\"", "&quot;", -1)
	msgStr = strings.ToLower(url.QueryEscape(msgStr))

	sign := msgStr + WlSmsPwd
	sign = fmt.Sprintf("%x", md5.Sum([]byte(sign)))
	param := WlSmsStruct{Uid: WlSmsSp, Sign: sign, Srcphone: WlSmsSrc, Msg: msgStr}
	data, _ := json.Marshal(param)
	fmt.Println(string(data))
	return string(data)
}

func (s *WlSms) saveData(b *BatchSms, msgId string) error {
	if len(*b) > 0 {
		for _, v := range *b {
			sp := SmsUpdate{ID: v.Id, MsgId: msgId}

			select {
			case wlSentSmsListChan <- sp:
				panellog.SmsPanelLog.Log("sendSms", "wl sms sent success and put in sent chan：", sp.ID)
			case <- time.After(time.Microsecond * 50):
				panellog.SmsPanelLog.Log("sendSms", "wl sms sent success but put in sent chan overtime：", sp.ID)
			}
		}
	}

	return nil
}