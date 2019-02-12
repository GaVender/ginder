package coroutinekit

import (
	"errors"
	"fmt"
	"github.com/xiaobai22/gokit-service/blackboardkit"
	"github.com/xiaobai22/gokit-service/httpkit"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/*
协程管理监控
实现goroutine运行情况监控等功能
适合常驻协程处理任务
不适合随时启动的短时间任务。

m := NewCoroutineKit()
m.Start(  "name", num, f(), panicRestart )


//如何知道协程退出，把函数包起来
func x(){
	f()
	//检测退出
}
还有要检测panic，如果panic可能也会退出

*/
const MAX_NUM = 100
const STATUS_INIT = 0
const STATUS_RUN = 1
const STATUS_OUT = 2
const STATUS_PANIC = 3

type CoroutineKit struct {
	mu          *sync.Mutex
	nodes       []*Node                      //每一组相同的goroutine占用一个node  主要作用可以按照启动的顺序展示监控信息
	nodeNames   map[string]*Node             //保存所有名称，用来去重
	historyInfo *blackboardkit.BlackBoradKit //记录历史的信息，例如之前已经完全退出的goroutine to do
}

func NewCoroutineKit() *CoroutineKit {
	ck := &CoroutineKit{}
	ck.nodes = make([]*Node, 0, 1000)
	ck.mu = &sync.Mutex{}
	ck.nodeNames = make(map[string]*Node)
	ck.historyInfo = blackboardkit.NewBlockBorad()
	ck.historyInfo.SetName("coroutinekit 历史信息")
	ck.historyInfo.Ready()
	return ck
}

//加入goroutine，1 名称，不要重复，重复会报错，2 启动多少个goroutine 3 执行函数  4 遇到panic后是否要重新启动
func (ck *CoroutineKit) Start(name string, num int, f func(), panicRestart bool) error {
	ck.mu.Lock()
	defer ck.mu.Unlock()
	name = strings.TrimSpace(name)
	//检查是否有重复名称
	_, ok := ck.nodeNames[name]
	if ok {
		return errors.New("duplicated name")
	}
	node := NewNode(ck, name, num, f, panicRestart)
	ck.nodes = append(ck.nodes, node)
	ck.nodeNames[name] = node
	node.Start() //启动
	return nil
}
func (ck *CoroutineKit) ShowAll() string {
	str := ""

	ck.mu.Lock()
	defer ck.mu.Unlock()
	for _, node := range ck.nodes {
		str += node.ShowAll()
	}
	return str
}
func (ck *CoroutineKit) Show(host string) string {
	str := ""
	str += "点击这里查看全部细节:\n任务列表:"
	str += `<a target="__blank" href="http://` + host + `/showall">click</a>`
	ck.mu.Lock()
	defer ck.mu.Unlock()
	for _, node := range ck.nodes {
		str += node.Show(host)
	}
	return str
}
func (ck *CoroutineKit) ShowDetail(nodename string) string {
	ck.mu.Lock()
	defer ck.mu.Unlock()
	node, ok := ck.nodeNames[nodename]
	if ok {
		return node.ShowDetails()
	}
	return "找不到信息"
}

type Node struct {
	name         string //coroutine名字, 如果没有名字可以填写""
	runnings     []*Routine
	f            func()
	panicRestart bool
	father       *CoroutineKit
	mu           *sync.Mutex
}

func NewNode(father *CoroutineKit, name string, num int, f func(), panicRestart bool) *Node {
	if num <= 0 {
		num = 1
	}
	if num > MAX_NUM {
		num = MAX_NUM
	}
	n := &Node{}
	n.name = name
	n.f = f
	n.panicRestart = panicRestart
	n.father = father
	n.mu = &sync.Mutex{}
	n.runnings = make([]*Routine, num, num)
	for i := 0; i < num; i++ {
		p := &Routine{}
		p.name = name
		p.startTime = ""
		p.endTime = ""
		p.panicTime = ""
		p.status = STATUS_INIT
		p.panicTimes = 0
		p.mu = &sync.Mutex{}
		p.lastPanicInfo = ""
		n.runnings[i] = p
	}
	return n
}

//展示日志信息，查看详情
func (n *Node) ShowDetails() string {
	str := ""
	n.mu.Lock()
	defer n.mu.Unlock()
	for k, v := range n.runnings {
		str += "------->\nGoroutine序号：" + strconv.Itoa(k)
		readme, _, _, _, _, _ := v.Show()
		str += readme
	}
	return "------------------" + n.name + "---------------------------->>\n" +
		str
}

//展示日志信息，查看汇总
func (n *Node) Show(host string) string {
	str := ""
	str1 := "正在运行的数量  :"
	str2 := "已经退出的数量  :"
	str3 := "已经panic的数量 :"
	str4 := "总数量          :"
	str5 := "panic历史数量  :"
	count1 := 0
	count2 := 0
	count3 := 0
	count4 := 0
	count5 := 0
	if len(host) >= 0 {
		str += "------->点击这里查看细节：\n" + `<a target="__blank" href="http://` + host + `/detail?name=` + n.name + `">click</a>`
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, v := range n.runnings {
		_, num1, num2, num3, num4, num5 := v.Show()
		count1 += num1
		count2 += num2
		count3 += num3
		count4 += num4
		count5 += num5
	}
	return "\n------------------" + n.name + "---------------------------->>\n" +
		str4 + strconv.Itoa(count4) + "\n" +
		str1 + strconv.Itoa(count1) + "\n" +
		str2 + strconv.Itoa(count2) + "\n" +
		str3 + strconv.Itoa(count3) + "\n" +
		str5 + strconv.Itoa(count5) + "\n" +
		str
}

func (n *Node) ShowAll() string {
	str := ""
	str1 := "正在运行的数量  :"
	str2 := "已经退出的数量  :"
	str3 := "已经panic的数量 :"
	str4 := "总数量          :"
	str5 := "panic历史数量   :"
	count1 := 0
	count2 := 0
	count3 := 0
	count4 := 0
	count5 := 0
	n.mu.Lock()
	defer n.mu.Unlock()
	for k, v := range n.runnings {
		str += "------->\nGoroutine序号：" + strconv.Itoa(k)
		readme, num1, num2, num3, num4, num5 := v.Show()
		str += readme
		count1 += num1
		count2 += num2
		count3 += num3
		count4 += num4
		count5 += num5

	}
	return "------------------" + n.name + "---------------------------->>\n" +
		str4 + strconv.Itoa(count4) + "\n" +
		str1 + strconv.Itoa(count1) + "\n" +
		str2 + strconv.Itoa(count2) + "\n" +
		str3 + strconv.Itoa(count3) + "\n" +
		str5 + strconv.Itoa(count5) + "\n" +
		str
}

func (n *Node) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	num := len(n.runnings)
	for i := 0; i < num; i++ {
		n.startOne(i)
	}
}
func (n *Node) startOne(goroutineNo int) {
	newf := func(no int) {
		defer func() {
			if co := recover(); co != nil {
				//检查panic
				str := fmt.Sprintln(co)
				n.SetPanic(no, str)
			}
		}()
		//开始运行
		n.SetRun(no)
		n.f()
		//检测退出
		n.SetOut(no)
	}
	go newf(goroutineNo)
}

//发生panic的时候
func (n *Node) SetPanic(no int, info string) {
	p := n.runnings[no]
	p.mu.Lock()
	defer p.mu.Unlock()
	atomic.AddUint64(&p.panicTimes, 1) //原子操作貌似是没有必要的
	p.lastPanicInfo = info
	p.status = STATUS_PANIC
	p.panicTime = time.Now().Format("2006-01-02 15:04:05")
	if n.panicRestart == true {
		time.Sleep(time.Millisecond * 100)
		n.startOne(no)
	}

}

//正常退出
func (n *Node) SetOut(no int) {
	p := n.runnings[no]
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = STATUS_OUT
	p.endTime = time.Now().Format("2006-01-02 15:04:05")
}

//开始运行
func (n *Node) SetRun(no int) {
	p := n.runnings[no]
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = STATUS_RUN
	p.startTime = time.Now().Format("2006-01-02 15:04:05")
	p.endTime = ""
}

type Routine struct {
	mu            *sync.Mutex
	name          string
	startTime     string
	endTime       string
	panicTime     string
	status        uint32 //0没有启动 1运行中 2退出 3panic
	panicTimes    uint64 //panic发生的次数
	lastPanicInfo string //最后一次panic的信息
}

//string信息收集  num1启动中的数量 num2已经退出的数量 num3已经panic的数量 num4 总数量  num5 历史panic数量
func (r *Routine) Show() (string, int, int, int, int, int) {
	str := ""
	num1 := 0
	num2 := 0
	num3 := 0
	num4 := 0
	num5 := 0
	r.mu.Lock()
	defer r.mu.Unlock()
	str += "\nGoroutine名称:" + r.name + "\n"
	statusReadme := ""
	if r.status == STATUS_INIT {
		statusReadme = "未启动"
	} else if r.status == STATUS_RUN {
		statusReadme = "运行中"
		num1 = 1
	} else if r.status == STATUS_OUT {
		statusReadme = "已退出"
		num2 = 1
	} else if r.status == STATUS_PANIC {
		statusReadme = "已恐慌"
		num3 = 1
	}
	num4 = 1
	num5 = int(r.panicTimes)
	str += "状态     :" + statusReadme + "\n"
	str += "启动时间  :" + r.startTime + "\n"
	str += "退出时间  :" + r.endTime + "\n"
	str += "异常时间  :" + r.panicTime + "\n"
	str += "异常次数  :" + strconv.FormatUint(r.panicTimes, 10) + "\n"
	str += "最后异常信息:" + r.lastPanicInfo + "\n"

	return str, num1, num2, num3, num4, num5
}

/**********************************************监控***************************************************/
func (ck *CoroutineKit) StartMonitor(port string) {
	go httpkit.NewSimpleHttpServer().Add("/", ck.httpShow).Add("/detail", ck.httpDetail).Add("/showall", ck.httpShowAll).Start(port)
}

func (ck *CoroutineKit) httpDetail(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("yes")
	r.ParseForm()
	name := r.FormValue("name")
	//fmt.Println(name)
	str := ck.ShowDetail(name)
	fmt.Fprintln(w, str)
}
func (ck *CoroutineKit) httpShow(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("yes")
	r.ParseForm()
	str := ck.Show(r.Host)
	str = strings.Replace(str, "\n", "<br/>", -1)
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	fmt.Fprintln(w, str)
}
func (ck *CoroutineKit) httpShowAll(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("yes")
	r.ParseForm()
	str := ck.ShowAll()
	fmt.Fprintln(w, str)
}
