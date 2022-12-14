package handlers

import (
	"fmt"
	"github.com/eatmoreapple/openwechat"
	util "github.com/hktalent/go-utils"
	"github.com/hktalent/wechatbot/config"
	"github.com/hktalent/wechatbot/service"
	"github.com/skip2/go-qrcode"
	"log"
	"runtime"
)

// MessageHandlerInterface 消息处理接口
type MessageHandlerInterface interface {
	handle(*openwechat.Message) error
	ReplyText(*openwechat.Message) error
}

type HandlerType string

const (
	GroupHandler = "group"
	UserHandler  = "user"
)

// QrCodeCallBack 登录扫码回调，
func QrCodeCallBack(uuid string) {
	if runtime.GOOS == "windows" {
		// 运行在Windows系统上
		openwechat.PrintlnQrcodeUrl(uuid)
	} else {
		log.Println("login in linux")
		q, _ := qrcode.New("https://login.weixin.qq.com/l/"+uuid, qrcode.Low)
		fmt.Println(q.ToString(true))
	}
}

// handlers 所有消息类型类型的处理器
var handlers map[HandlerType]MessageHandlerInterface
var UserService service.UserServiceInterface

var EanbleGroup = true

func init() {
	util.RegInitFunc(func() {
		handlers = make(map[HandlerType]MessageHandlerInterface)
		handlers[GroupHandler] = NewGroupMessageHandler()
		handlers[UserHandler] = NewUserMessageHandler()
		UserService = service.NewUserService()
		EanbleGroup = util.GetValAsBool("EanbleGroup")
	})

}

// Handler 全局处理入口
func Handler(msg *openwechat.Message) {
	log.Printf("hadler Received msg : %v", msg.Content)
	// 处理群消息
	if EanbleGroup && msg.IsSendByGroup() {
		handlers[GroupHandler].handle(msg)
		return
	}

	// 好友申请
	if msg.IsFriendAdd() {
		if config.LoadConfig().AutoPass {
			_, err := msg.Agree("???")
			if err != nil {
				log.Fatalf("add friend agree error : %v", err)
				return
			}
		}
	}

	// 私聊
	handlers[UserHandler].handle(msg)
}
