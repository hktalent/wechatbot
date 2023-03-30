package handlers

import (
	"fmt"
	"github.com/eatmoreapple/openwechat"
	util "github.com/hktalent/go-utils"
	"github.com/hktalent/wechatbot/gtp"
	"log"
	"strings"
)

var _ MessageHandlerInterface = (*UserMessageHandler)(nil)

// UserMessageHandler 私聊消息处理
type UserMessageHandler struct {
}

// handle 处理消息
func (g *UserMessageHandler) handle(msg *openwechat.Message) error {
	if msg.IsText() {
		return g.ReplyText(msg)
	}
	return nil
}

// NewUserMessageHandler 创建私聊处理器
func NewUserMessageHandler() MessageHandlerInterface {
	return &UserMessageHandler{}
}

// ReplyText 发送文本消息到群
func (g *UserMessageHandler) ReplyText(msg *openwechat.Message) error {
	// 接收私聊消息
	sender, err := msg.Sender()
	log.Printf("Received User %v Text Msg : %v", sender.NickName, msg.Content)
	if UserService.ClearUserSessionContext(sender.ID(), msg.Content) {
		_, err = msg.ReplyText("上下文已经清空了，你可以问下一个问题啦。")
		if err != nil {
			log.Printf("response user error: %v \n", err)
		}
		return nil
	}
	s1 := util.GetVal("UseSelf")

	if "" != s1 && (-1 == strings.Index(s1, sender.NickName) || msg.FromUserName != msg.ToUserName) {
		return nil
	}

	// 获取上下文，向GPT发起请求
	requestText := strings.TrimSpace(msg.Content)
	requestText = strings.Trim(msg.Content, "\n")

	requestText = UserService.GetUserSessionContext(sender.ID()) + requestText
	reply, err := gtp.Completions(requestText)
	if err != nil {
		log.Printf("gtp request error: %v \n", err)
		msg.ReplyText(util.GetVal("ErrMsg") + fmt.Sprintf("\n%v\n", err))
		return err
	}
	if reply == "" {
		return nil
	}

	// 设置上下文，回复用户
	reply = strings.TrimSpace(reply)
	reply = strings.Trim(reply, "\n")
	UserService.SetUserSessionContext(sender.ID(), requestText, reply)
	//if EanbleGroup {
	//	reply = "系统自动回复：\n" + reply
	//}
	_, err = msg.ReplyText(reply)
	if err != nil {
		log.Printf("response user error: %v \n", err)
	}
	return err
}
