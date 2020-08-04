package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Logiase/gomirai/bot"
	"github.com/Logiase/gomirai/message"
)

type KEY struct {
	word string
	reply string
}

type KEYS struct {
	data []KEY
}

type COUNTER struct {
	ID uint
	REMARK string
	BEGIN time.Time
	COUNT []time.Time
}

const (
	qq uint = 2540719484
	helpKey = "help"
	execKey = "exec"
	colorKey = "color"
	tpinKey = "tpin"
	counterKey = "cod"
	help = `Starx's bot - command list
获取此帮助: /help
执行终端命令: /exec [cmd] (admin only)
来一份色图: /color
获取TCP延迟: /tpin [ip,hostname] [port]
启动计时器: /cod [start,stop,count] [remark]
`
	//wordsPath = "words.json"
)

var (
	master = []uint {1787074172}
	//
	counter []COUNTER
	//
	global KEYS
	//unknow KEYS
)

func isTrusted(id uint) bool{
	for _,p := range master {
		if p == id{
			return true
		}
	}
	return false
}

func getCounted(id uint) (COUNTER,int){
	for i,p := range counter{
		if p.ID == id{
			return p,i
		}
	}
	return COUNTER{
		ID:     0,
		REMARK: "",
		BEGIN:  time.Time{},
		COUNT:  []time.Time{},
	},-1
}

func delCounted(i int) {
	counter = append(counter[:i],counter[i+1:]...)
}

func getARG(full string,key string)([]string,error){
	key = "/" + key
	if !strings.HasPrefix(full,key){
		return []string{},errors.New("获取命令参数错误: 未知的命令")
	}
	return strings.Fields(strings.TrimSpace(strings.Trim(full,key)) ),nil
}

func safeExec(ucmd string,group uint,id uint,b *bot.Bot) {
	if len(ucmd) == 0 {
		_,_ = b.SendGroupMessage(group,0,message.AtMessage(id),message.PlainMessage("命令执行参数不完整"))
		return
	}
	if isTrusted(id){
		cmd := strings.Fields(ucmd)
		app := cmd[0]
		args := cmd[1:]
		out,_ := exec.Command(app,args...).Output()
		_,_ = b.SendGroupMessage(group,0,message.AtMessage(id),message.PlainMessage(strings.TrimSpace(string(out))))
	} else {
		_,_ = b.SendGroupMessage(group,0,message.AtMessage(id),message.PlainMessage("命令执行未受信任"))
	}

}

func tcping(pat []string,group uint,id uint,b *bot.Bot) {
	// ip/host [0]
	// port [1]
	args := len(pat)
	if args == 0 || args < 2{
		_,_ = b.SendGroupMessage(group,0,message.AtMessage(id),message.PlainMessage("TCPING 参数不完整"))
		return
	}
	addr := net.ParseIP(pat[0])
	if addr == nil {
		addr_,err := net.LookupIP(pat[0])
		if err != nil{
			_,_ = b.SendGroupMessage(group,0,message.AtMessage(id),message.PlainMessage("未找到DNS记录"))
			return
		}
		addr = addr_[0]
	}
	start := time.Now()
	var conn net.Conn
	var err error
	if !strings.Contains(addr.String(),":"){
		conn,err = net.Dial("tcp",addr.String()+":"+pat[1])
	} else {
		conn,err = net.Dial("tcp","[" + addr.String() + "]" + ":" +pat[1])
	}

	if err != nil {
		_,_ = b.SendGroupMessage(group,0,message.AtMessage(id),message.PlainMessage("建立TCP连接时出错: "+err.Error()))
		return
	}
	_,_ = b.SendGroupMessage(group,0,message.AtMessage(id),message.PlainMessage(strings.Join([]string{"TCPING",pat[0],":",pat[1],time.Now().Sub(start).String()}," ")))
	_ = conn.Close()

}

func count(pat []string,group uint,id uint,b *bot.Bot){
	if len(pat) == 0{
		_, _ = b.SendGroupMessage(group, 0, message.AtMessage(id), message.PlainMessage("计时器参数不完整"))
		return
	}
	switch pat[0] {
	case "start":
		var remark string
		if len(pat) == 1{
			remark = "未命名记录"
		} else {
			remark = strings.Join(pat[1:]," ")
		}

		counter = append(counter, COUNTER{
				ID:     id,
				REMARK: remark,
				BEGIN:  time.Now(),
				COUNT:  nil,
			})
		_, _ = b.SendGroupMessage(group, 0, message.AtMessage(id), message.PlainMessage(remark+"\n已开始计时"))
	case "stop":
		endTime := time.Now()
		_,index := getCounted(id)
		if index != -1 {
			counts := len(counter[index].COUNT)
			if counts <= 1 {
				_, _ = b.SendGroupMessage(group, 0, message.AtMessage(id), message.PlainMessage(counter[index].REMARK+"\n计时: "+endTime.Sub(counter[index].BEGIN).String()))
				delCounted(index)
			} else {
				var rp string
				for i,cts := range counter[index].COUNT{
					rp += "计次: " + strconv.Itoa(i+1)+"\n计时: "+ cts.Sub(counter[index].BEGIN).String() + "\n"
				}
				_, _ = b.SendGroupMessage(group, 0, message.AtMessage(id), message.PlainMessage(counter[index].REMARK + "\n" + rp))
				delCounted(index)
			}
		} else {
			_, _ = b.SendGroupMessage(group, 0, message.AtMessage(id), message.PlainMessage("未找到你的计时记录"))
		}
	case "count":
		endTime := time.Now()
		_,index := getCounted(id)
		if index != -1 {
			counter[index].COUNT=append(counter[index].COUNT,endTime)
			_, _ = b.SendGroupMessage(group, 0, message.AtMessage(id), message.PlainMessage(counter[index].REMARK + "\n计次" +strconv.Itoa(len(counter[index].COUNT))+"已开始"))
		} else {
			_, _ = b.SendGroupMessage(group, 0, message.AtMessage(id), message.PlainMessage("未找到你的计时记录"))
		}
	default:
		_, _ = b.SendGroupMessage(group, 0, message.AtMessage(id), message.PlainMessage("未知的计时器命令"))
	}
}

func findkey(full string,group uint,id uint,b *bot.Bot){
	var matchs []int
	for i,sw := range global.data{
		if strings.Contains(full,sw.word){
			matchs = append(matchs,i)
		}
	}
	matchsN := len(matchs)
	// 3
	if matchsN == 1{
		_, _ = b.SendGroupMessage(group, 0, message.AtMessage(id), message.PlainMessage(global.data[0].reply))
	}else if matchsN > 1{
		_, _ = b.SendGroupMessage(group, 0, message.AtMessage(id), message.PlainMessage(global.data[rand.Intn(matchsN)].reply))
	}
	//else {
	//	_, _ = b.SendGroupMessage(group, 0, message.AtMessage(id), message.PlainMessage("words not found"))
	//}
}

func main() {
	global = KEYS{data: []KEY{{
		word:  "测试",
		reply: "测试成功啦",
	}, {
		word:  "测试",
		reply: "测试2也成功啦",
	}, {
		word:  "测试",
		reply: "随机测试啦",
	}, {
		word:  "测试",
		reply: "给我滚去吃饭!!",
	}}}
	// Catch interrupt
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	// Create a bot instance
	c := bot.NewClient("default", "http://192.168.31.99:8888", "starxiyf")
	// Setup Log Level
	c.Logger.Level = logrus.TraceLevel
	// If able to connect to api
	key, err := c.Auth()
	if err != nil {
		c.Logger.Fatal(err)
	}
	// Check if id is mismatch
	b, err := c.Verify(qq, key)
	if err != nil {
		c.Logger.Fatal(err)
	}
	//defer c.Release(qq)

	go func() {
		err = b.FetchMessages()
		if err != nil {
			c.Logger.Fatal(err)
		}
	}()

	for {
		select {
		case e := <-b.Chan:
			switch e.Type {
			case message.EventReceiveGroupMessage:
				quote := e.MessageChain[0].Id
				var text string
				for _,mp := range e.MessageChain{
					pt := mp.Text
					if len(pt) != 0 {
						text += pt
					}
				}
				//fmt.Printf("%+v\n", e.MessageChain)
				if len(text) != 0 {
					// HELP
					_,errHelp := getARG(text,helpKey)
					if errHelp == nil {
						_,_ = b.SendGroupMessage(e.Sender.Group.Id,quote,message.PlainMessage(strings.TrimSpace(help)))
						break
					}
					// EXEC
					argsExec,errExec := getARG(text,execKey)
					if errExec == nil {
						go safeExec(strings.Join(argsExec," ") ,e.Sender.Group.Id,e.Sender.Id,b)
						break
					}
					// Colorful pic
					_, errColor := getARG(text, colorKey)
					if errColor == nil {
						_,_ = b.SendGroupMessage(e.Sender.Group.Id,quote,message.ImageMessage("url","https://i.xinger.ink:4443/images.php"))
						break
					}
					// TCPING
					argsTpin,errTpin := getARG(text,tpinKey)
					if errTpin == nil {
						go tcping(argsTpin,e.Sender.Group.Id,e.Sender.Id,b)
						break
					}
					// COUNTER
					argsCoun,errCoun := getARG(text,counterKey)
					if errCoun == nil {
						go count(argsCoun,e.Sender.Group.Id,e.Sender.Id,b)
						break
					}
					// KEYWORD TODO
					go findkey(text,e.Sender.Group.Id,e.Sender.Id,b)
					}
			}
		case <-interrupt:
			fmt.Println("######")
			fmt.Println("interrupt")
			fmt.Println("######")
			//c.Release(qq)
			_ = c.Release(qq)
			return
		}

	}
}