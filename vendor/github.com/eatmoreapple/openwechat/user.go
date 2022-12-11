package openwechat

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// User 抽象的用户结构: 好友 群组 公众号
type User struct {
	HideInputBarFlag  int
	StarFriend        int
	Sex               int
	AppAccountFlag    int
	VerifyFlag        int
	ContactFlag       int
	WebWxPluginSwitch int
	HeadImgFlag       int
	SnsFlag           int
	IsOwner           int
	MemberCount       int
	ChatRoomId        int
	UniFriend         int
	OwnerUin          int
	Statues           int
	AttrStatus        int64
	Uin               int64
	Province          string
	City              string
	Alias             string
	DisplayName       string
	KeyWord           string
	EncryChatRoomId   string
	UserName          string
	NickName          string
	HeadImgUrl        string
	RemarkName        string
	PYInitial         string
	PYQuanPin         string
	RemarkPYInitial   string
	RemarkPYQuanPin   string
	Signature         string

	MemberList Members

	Self *Self
}

// implement fmt.Stringer
func (u *User) String() string {
	return fmt.Sprintf("<User:%s>", u.NickName)
}

// GetAvatarResponse 获取用户头像
func (u *User) GetAvatarResponse() (*http.Response, error) {
	return u.Self.Bot.Caller.Client.WebWxGetHeadImg(u)
}

// SaveAvatar 下载用户头像
func (u *User) SaveAvatar(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return u.SaveAvatarWithWriter(file)
}

func (u *User) SaveAvatarWithWriter(writer io.Writer) error {
	resp, err := u.GetAvatarResponse()
	if err != nil {
		return err
	}
	// 这里获取头像的响应有时可能会异常
	// 一般为网路原因
	// 再去请求一次即可解决
	if resp.ContentLength == 0 && resp.Header.Get("Content-Type") == "image/jpeg" {
		resp, err = u.GetAvatarResponse()
		if err != nil {
			return err
		}
	}
	// 写文件前判断下 content length 是否是 0，不然保存的头像会出现
	// image not loaded  try to open it externally to fix format problem 问题
	if resp.ContentLength == 0 {
		return fmt.Errorf("get avatar response content length is 0")
	}
	defer resp.Body.Close()
	_, err = io.Copy(writer, resp.Body)
	return err
}

// Detail 获取用户的详情
func (u *User) Detail() error {
	if u.UserName == u.Self.UserName {
		return nil
	}
	members := Members{u}
	request := u.Self.Bot.Storage.Request
	newMembers, err := u.Self.Bot.Caller.WebWxBatchGetContact(members, request)
	if err != nil {
		return err
	}
	newMembers.init(u.Self)
	user := newMembers.First()
	*u = *user
	u.MemberList.init(u.Self)
	return nil
}

// IsFriend 判断是否为好友
func (u *User) IsFriend() bool {
	return !u.IsGroup() && strings.HasPrefix(u.UserName, "@") && u.VerifyFlag == 0
}

// IsGroup 判断是否为群组
func (u *User) IsGroup() bool {
	return strings.HasPrefix(u.UserName, "@@") && u.VerifyFlag == 0
}

// IsMP  判断是否为公众号
func (u *User) IsMP() bool {
	return u.VerifyFlag == 8 || u.VerifyFlag == 24 || u.VerifyFlag == 136
}

// Pin 将联系人置顶
func (u *User) Pin() error {
	req := u.Self.Bot.Storage.Request
	return u.Self.Bot.Caller.WebWxRelationPin(req, u, 1)
}

// UnPin 将联系人取消置顶
func (u *User) UnPin() error {
	req := u.Self.Bot.Storage.Request
	return u.Self.Bot.Caller.WebWxRelationPin(req, u, 0)
}

// IsPin 判断当前联系人(好友、群组、公众号)是否为置顶状态
func (u *User) IsPin() bool {
	return u.ContactFlag == 2051
}

// ID 获取用户的唯一标识 只对当前登录的用户有效
// ID 和 UserName 的区别是 ID 多次登录不会变化，而 UserName 只针对当前登录会话有效
func (u *User) ID() string {
	// 首先尝试获取uid
	if u.Uin != 0 {
		return strconv.FormatInt(u.Uin, 10)
	}
	// 如果uid不存在，尝试从头像url中获取
	if u.HeadImgUrl != "" {
		index := strings.Index(u.HeadImgUrl, "?") + 1
		if len(u.HeadImgUrl) > index {
			query := u.HeadImgUrl[index:]
			params, err := url.ParseQuery(query)
			if err != nil {
				return ""
			}
			return params.Get("seq")

		}
	}
	return ""
}

// 格式化emoji表情
func (u *User) formatEmoji() {
	u.NickName = FormatEmoji(u.NickName)
	u.RemarkName = FormatEmoji(u.RemarkName)
	u.DisplayName = FormatEmoji(u.DisplayName)
}

// Self 自己,当前登录用户对象
type Self struct {
	*User
	Bot        *Bot
	fileHelper *Friend
	members    Members
	friends    Friends
	groups     Groups
	mps        Mps
}

// Members 获取所有的好友、群组、公众号信息
func (s *Self) Members(update ...bool) (Members, error) {
	// 首先判断缓存里有没有,如果没有则去更新缓存
	// 判断是否需要更新,如果传入的参数不为nil,则取第一个
	if s.members == nil || (len(update) > 0 && update[0]) {
		if err := s.updateMembers(); err != nil {
			return nil, err
		}
	}
	return s.members, nil
}

// 更新联系人处理
func (s *Self) updateMembers() error {
	info := s.Bot.Storage.LoginInfo
	members, err := s.Bot.Caller.WebWxGetContact(info)
	if err != nil {
		return err
	}
	members.init(s)
	s.members = members
	return nil
}

// FileHelper 获取文件传输助手对象，封装成Friend返回
//
//	fh, err := self.FileHelper() // or fh := openwechat.NewFriendHelper(self)
func (s *Self) FileHelper() (*Friend, error) {
	// 如果缓存里有，直接返回，否则去联系人里面找
	if s.fileHelper != nil {
		return s.fileHelper, nil
	}
	members, err := s.Members()
	if err != nil {
		return nil, err
	}
	users := members.SearchByUserName(1, "filehelper")
	if users == nil {
		s.fileHelper = NewFriendHelper(s)
	} else {
		s.fileHelper = &Friend{users.First()}
	}
	return s.fileHelper, nil
}

// Friends 获取所有的好友
func (s *Self) Friends(update ...bool) (Friends, error) {
	if s.friends == nil || (len(update) > 0 && update[0]) {
		if _, err := s.Members(true); err != nil {
			return nil, err
		}
		s.friends = s.members.Friends()
	}
	return s.friends, nil
}

// Groups 获取所有的群组
func (s *Self) Groups(update ...bool) (Groups, error) {
	if s.groups == nil || (len(update) > 0 && update[0]) {
		if _, err := s.Members(true); err != nil {
			return nil, err
		}
		s.groups = s.members.Groups()
	}
	return s.groups, nil
}

// Mps 获取所有的公众号
func (s *Self) Mps(update ...bool) (Mps, error) {
	if s.mps == nil || (len(update) > 0 && update[0]) {
		if _, err := s.Members(true); err != nil {
			return nil, err
		}
		s.mps = s.members.MPs()
	}
	return s.mps, nil
}

// UpdateMembersDetail 更新所有的联系人信息
func (s *Self) UpdateMembersDetail() error {
	// 先获取所有的联系人
	members, err := s.Members()
	if err != nil {
		return err
	}
	return members.detail(s)
}

func (s *Self) sendTextToUser(user *User, text string) (*SentMessage, error) {
	msg := NewTextSendMessage(text, s.UserName, user.UserName)
	msg.FromUserName = s.UserName
	msg.ToUserName = user.UserName
	info := s.Bot.Storage.LoginInfo
	request := s.Bot.Storage.Request
	sentMessage, err := s.Bot.Caller.WebWxSendMsg(msg, info, request)
	return s.sendMessageWrapper(sentMessage, err)
}

func (s *Self) sendImageToUser(user *User, file *os.File) (*SentMessage, error) {
	req := s.Bot.Storage.Request
	info := s.Bot.Storage.LoginInfo
	sentMessage, err := s.Bot.Caller.WebWxSendImageMsg(file, req, info, s.UserName, user.UserName)
	return s.sendMessageWrapper(sentMessage, err)
}

func (s *Self) sendVideoToUser(user *User, file *os.File) (*SentMessage, error) {
	req := s.Bot.Storage.Request
	info := s.Bot.Storage.LoginInfo
	sentMessage, err := s.Bot.Caller.WebWxSendVideoMsg(file, req, info, s.UserName, user.UserName)
	return s.sendMessageWrapper(sentMessage, err)
}

func (s *Self) sendFileToUser(user *User, file *os.File) (*SentMessage, error) {
	req := s.Bot.Storage.Request
	info := s.Bot.Storage.LoginInfo
	sentMessage, err := s.Bot.Caller.WebWxSendFile(file, req, info, s.UserName, user.UserName)
	return s.sendMessageWrapper(sentMessage, err)
}

// SendTextToFriend 发送文本消息给好友
func (s *Self) SendTextToFriend(friend *Friend, text string) (*SentMessage, error) {
	return s.sendTextToUser(friend.User, text)
}

// SendImageToFriend 发送图片消息给好友
func (s *Self) SendImageToFriend(friend *Friend, file *os.File) (*SentMessage, error) {
	return s.sendImageToUser(friend.User, file)
}

// SendVideoToFriend 发送视频给好友
func (s *Self) SendVideoToFriend(friend *Friend, file *os.File) (*SentMessage, error) {
	return s.sendVideoToUser(friend.User, file)
}

// SendFileToFriend 发送文件给好友
func (s *Self) SendFileToFriend(friend *Friend, file *os.File) (*SentMessage, error) {
	return s.sendFileToUser(friend.User, file)
}

// SetRemarkNameToFriend 设置好友备注
//
//	self.SetRemarkNameToFriend(friend, "remark") // or friend.SetRemarkName("remark")
func (s *Self) SetRemarkNameToFriend(friend *Friend, remarkName string) error {
	req := s.Bot.Storage.Request
	return s.Bot.Caller.WebWxOplog(req, remarkName, friend.UserName)
}

// CreateGroup 创建群聊
// topic 群昵称,可以传递字符串
// friends 群员,最少为2个，加上自己3个,三人才能成群
func (s *Self) CreateGroup(topic string, friends ...*Friend) (*Group, error) {
	if len(friends) < 2 {
		return nil, errors.New("a group must be at least 2 members")
	}
	req := s.Bot.Storage.Request
	info := s.Bot.Storage.LoginInfo
	group, err := s.Bot.Caller.WebWxCreateChatRoom(req, info, topic, friends)
	if err != nil {
		return nil, err
	}
	group.Self = s
	err = group.Detail()
	return group, err
}

// AddFriendsIntoGroup 拉多名好友进群
// 最好自己是群主,成功率高一点,因为有的群允许非群组拉人,而有的群不允许
func (s *Self) AddFriendsIntoGroup(group *Group, friends ...*Friend) error {
	if len(friends) == 0 {
		return nil
	}
	// 获取群的所有的群员
	groupMembers, err := group.Members()
	if err != nil {
		return err
	}
	// 判断当前的成员在不在群里面
	for _, friend := range friends {
		for _, member := range groupMembers {
			if member.UserName == friend.UserName {
				return fmt.Errorf("user %s has alreay in this group", friend.String())
			}
		}
	}
	req := s.Bot.Storage.Request
	info := s.Bot.Storage.LoginInfo
	return s.Bot.Caller.AddFriendIntoChatRoom(req, info, group, friends...)
}

// RemoveMemberFromGroup 从群聊中移除用户
// Deprecated
// 无论是网页版，还是程序上都不起作用
func (s *Self) RemoveMemberFromGroup(group *Group, members Members) error {
	if len(members) == 0 {
		return nil
	}
	if group.IsOwner == 0 {
		return errors.New("group owner required")
	}
	groupMembers, err := group.Members()
	if err != nil {
		return err
	}
	// 判断用户是否在群聊中
	var count int
	for _, member := range members {
		for _, gm := range groupMembers {
			if gm.UserName == member.UserName {
				count++
			}
		}
	}
	if count != len(members) {
		return errors.New("invalid members")
	}
	req := s.Bot.Storage.Request
	info := s.Bot.Storage.LoginInfo
	return s.Bot.Caller.RemoveFriendFromChatRoom(req, info, group, members...)
}

// AddFriendIntoManyGroups 拉好友进多个群聊
// AddFriendIntoGroups, 名字和上面的有点像
func (s *Self) AddFriendIntoManyGroups(friend *Friend, groups ...*Group) error {
	for _, group := range groups {
		if err := s.AddFriendsIntoGroup(group, friend); err != nil {
			return err
		}
	}
	return nil
}

// RenameGroup 群组重命名
func (s *Self) RenameGroup(group *Group, newName string) error {
	req := s.Bot.Storage.Request
	info := s.Bot.Storage.LoginInfo
	return s.Bot.Caller.WebWxRenameChatRoom(req, info, newName, group)
}

// SendTextToGroup 发送文本消息给群组
func (s *Self) SendTextToGroup(group *Group, text string) (*SentMessage, error) {
	return s.sendTextToUser(group.User, text)
}

// SendImageToGroup 发送图片消息给群组
func (s *Self) SendImageToGroup(group *Group, file *os.File) (*SentMessage, error) {
	return s.sendImageToUser(group.User, file)
}

// SendVideoToGroup 发送视频给群组
func (s *Self) SendVideoToGroup(group *Group, file *os.File) (*SentMessage, error) {
	return s.sendVideoToUser(group.User, file)
}

// SendFileToGroup 发送文件给群组
func (s *Self) SendFileToGroup(group *Group, file *os.File) (*SentMessage, error) {
	return s.sendFileToUser(group.User, file)
}

// RevokeMessage 撤回消息
//
//	sentMessage, err := friend.SendText("message")
//	if err == nil {
//	    self.RevokeMessage(sentMessage) // or sentMessage.Revoke()
//	}
func (s *Self) RevokeMessage(msg *SentMessage) error {
	return s.Bot.Caller.WebWxRevokeMsg(msg, s.Bot.Storage.Request)
}

// 转发消息接口
func (s *Self) forwardMessage(msg *SentMessage, delay time.Duration, users ...*User) error {
	info := s.Bot.Storage.LoginInfo
	req := s.Bot.Storage.Request
	switch msg.Type {
	case MsgTypeText:
		for _, user := range users {
			msg.FromUserName = s.UserName
			msg.ToUserName = user.UserName
			if _, err := s.Self.Bot.Caller.WebWxSendMsg(msg.SendMessage, info, req); err != nil {
				return err
			}
			time.Sleep(delay)
		}
	case MsgTypeImage:
		for _, user := range users {
			msg.FromUserName = s.UserName
			msg.ToUserName = user.UserName
			if _, err := s.Self.Bot.Caller.Client.WebWxSendMsgImg(msg.SendMessage, req, info); err != nil {
				return err
			}
			time.Sleep(delay)
		}
	case AppMessage:
		for _, user := range users {
			msg.FromUserName = s.UserName
			msg.ToUserName = user.UserName
			if _, err := s.Self.Bot.Caller.Client.WebWxSendAppMsg(msg.SendMessage, req); err != nil {
				return err
			}
			time.Sleep(delay)
		}
	default:
		return fmt.Errorf("unsupported message type: %s", msg.Type)
	}
	return nil
}

// ForwardMessageToFriends 转发给好友
func (s *Self) ForwardMessageToFriends(msg *SentMessage, delay time.Duration, friends ...*Friend) error {
	members := Friends(friends).AsMembers()
	return s.forwardMessage(msg, delay, members...)
}

// ForwardMessageToGroups 转发给群组
func (s *Self) ForwardMessageToGroups(msg *SentMessage, delay time.Duration, groups ...*Group) error {
	members := Groups(groups).AsMembers()
	return s.forwardMessage(msg, delay, members...)
}

// sendTextToMembers 发送文本消息给群组或者好友
func (s *Self) sendTextToMembers(text string, delay time.Duration, members ...*User) error {
	if len(members) == 0 {
		return nil
	}
	user := members[0]
	msg, err := s.sendTextToUser(user, text)
	if err != nil {
		return err
	}
	time.Sleep(delay)
	return s.forwardMessage(msg, delay, members[1:]...)
}

// sendImageToMembers 发送图片消息给群组或者好友
func (s *Self) sendImageToMembers(img *os.File, delay time.Duration, members ...*User) error {
	if len(members) == 0 {
		return nil
	}
	user := members[0]
	msg, err := s.sendImageToUser(user, img)
	if err != nil {
		return err
	}
	time.Sleep(delay)
	return s.forwardMessage(msg, delay, members[1:]...)
}

// sendVideoToMembers 发送视频消息给群组或者好友
func (s *Self) sendVideoToMembers(video *os.File, delay time.Duration, members ...*User) error {
	if len(members) == 0 {
		return nil
	}
	user := members[0]
	msg, err := s.sendVideoToUser(user, video)
	if err != nil {
		return err
	}
	time.Sleep(delay)
	return s.forwardMessage(msg, delay, members[1:]...)
}

func (s *Self) sendFileToMembers(file *os.File, delay time.Duration, members ...*User) error {
	if len(members) == 0 {
		return nil
	}
	user := members[0]
	msg, err := s.sendFileToUser(user, file)
	if err != nil {
		return err
	}
	time.Sleep(delay)
	return s.forwardMessage(msg, delay, members[1:]...)
}

// SendTextToFriends 发送文本消息给好友
func (s *Self) SendTextToFriends(text string, delay time.Duration, friends ...*Friend) error {
	members := Friends(friends).AsMembers()
	return s.sendTextToMembers(text, delay, members...)
}

// SendImageToFriends 发送图片消息给好友
func (s *Self) SendImageToFriends(img *os.File, delay time.Duration, friends ...*Friend) error {
	members := Friends(friends).AsMembers()
	return s.sendImageToMembers(img, delay, members...)
}

// SendFileToFriends 发送文件给好友
func (s *Self) SendFileToFriends(file *os.File, delay time.Duration, friends ...*Friend) error {
	members := Friends(friends).AsMembers()
	return s.sendFileToMembers(file, delay, members...)
}

// SendVideoToFriends 发送视频给好友
func (s *Self) SendVideoToFriends(video *os.File, delay time.Duration, friends ...*Friend) error {
	members := Friends(friends).AsMembers()
	return s.sendVideoToMembers(video, delay, members...)
}

// SendTextToGroups 发送文本消息给群组
func (s *Self) SendTextToGroups(text string, delay time.Duration, groups ...*Group) error {
	members := Groups(groups).AsMembers()
	return s.sendTextToMembers(text, delay, members...)
}

// SendImageToGroups 发送图片消息给群组
func (s *Self) SendImageToGroups(img *os.File, delay time.Duration, groups ...*Group) error {
	members := Groups(groups).AsMembers()
	return s.sendImageToMembers(img, delay, members...)
}

// SendFileToGroups 发送文件给群组
func (s *Self) SendFileToGroups(file *os.File, delay time.Duration, groups ...*Group) error {
	members := Groups(groups).AsMembers()
	return s.sendFileToMembers(file, delay, members...)
}

// SendVideoToGroups 发送视频给群组
func (s *Self) SendVideoToGroups(video *os.File, delay time.Duration, groups ...*Group) error {
	members := Groups(groups).AsMembers()
	return s.sendVideoToMembers(video, delay, members...)
}

// Members 抽象的用户组
type Members []*User

// Count 统计数量
func (m Members) Count() int {
	return len(m)
}

// First 获取第一个
func (m Members) First() *User {
	if m.Count() > 0 {
		u := m[0]
		return u
	}
	return nil
}

// Last 获取最后一个
func (m Members) Last() *User {
	if m.Count() > 0 {
		u := m[m.Count()-1]
		return u
	}
	return nil
}

// SearchByUserName 根据用户名查找
func (m Members) SearchByUserName(limit int, username string) (results Members) {
	return m.Search(limit, func(user *User) bool { return user.UserName == username })
}

// SearchByNickName 根据昵称查找
func (m Members) SearchByNickName(limit int, nickName string) (results Members) {
	return m.Search(limit, func(user *User) bool { return user.NickName == nickName })
}

// SearchByRemarkName 根据备注查找
func (m Members) SearchByRemarkName(limit int, remarkName string) (results Members) {
	return m.Search(limit, func(user *User) bool { return user.RemarkName == remarkName })
}

// Search 根据自定义条件查找
func (m Members) Search(limit int, searchFuncList ...func(user *User) bool) (results Members) {
	return search(m, limit, func(group *User) bool {
		for _, searchFunc := range searchFuncList {
			if !searchFunc(group) {
				return false
			}
		}
		return true
	})
}

// GetByUserName 根据username查找用户
func (m Members) GetByUserName(username string) (*User, bool) {
	users := m.SearchByUserName(1, username)
	user := users.First()
	return user, user != nil
}

// GetByRemarkName 根据remarkName查找用户
func (m Members) GetByRemarkName(remarkName string) (*User, bool) {
	users := m.SearchByRemarkName(1, remarkName)
	user := users.First()
	return user, user != nil
}

// GetByNickName 根据nickname查找用户
func (m Members) GetByNickName(nickname string) (*User, bool) {
	users := m.SearchByNickName(1, nickname)
	user := users.First()
	return user, user != nil
}

func (m Members) Friends() Friends {
	friends := make(Friends, 0)
	for _, mb := range m {
		if mb.IsFriend() {
			friend := &Friend{mb}
			friends = append(friends, friend)
		}
	}
	return friends
}

func (m Members) Groups() Groups {
	groups := make(Groups, 0)
	for _, mb := range m {
		if mb.IsGroup() {
			group := &Group{mb}
			groups = append(groups, group)
		}
	}
	return groups
}

func (m Members) MPs() Mps {
	mps := make(Mps, 0)
	for _, mb := range m {
		if mb.IsMP() {
			mp := &Mp{mb}
			mps = append(mps, mp)
		}
	}
	return mps
}

// 获取当前Members的详情
func (m Members) detail(self *Self) error {
	// 获取他们的数量
	members := m

	count := members.Count()
	// 一次更新50个,分情况讨论

	// 获取总的需要更新的次数
	var times int
	if count < 50 {
		times = 1
	} else {
		times = count / 50
	}
	var newMembers Members
	request := self.Bot.Storage.Request
	var pMembers Members
	// 分情况依次更新
	for i := 1; i <= times; i++ {
		if times == 1 {
			pMembers = members
		} else {
			pMembers = members[(i-1)*50 : i*50]
		}
		nMembers, err := self.Bot.Caller.WebWxBatchGetContact(pMembers, request)
		if err != nil {
			return err
		}
		newMembers = append(newMembers, nMembers...)
	}
	// 最后判断是否全部更新完毕
	total := times * 50
	if total < count {
		// 将全部剩余的更新完毕
		left := count - total
		pMembers = members[total : total+left]
		nMembers, err := self.Bot.Caller.WebWxBatchGetContact(pMembers, request)
		if err != nil {
			return err
		}
		newMembers = append(newMembers, nMembers...)
	}
	if len(newMembers) > 0 {
		newMembers.init(self)
		self.members = newMembers
	}
	return nil
}

func (m Members) init(self *Self) {
	for _, member := range m {
		member.Self = self
		member.formatEmoji()
	}
}

func newFriend(username string, self *Self) *Friend {
	return &Friend{&User{UserName: username, Self: self}}
}

// NewFriendHelper 这里为了兼容Desktop版本找不到文件传输助手的问题
// 文件传输助手的微信身份标识符永远是filehelper
// 这种形式的对象可能缺少一些其他属性
// 但是不影响发送信息的功能
func NewFriendHelper(self *Self) *Friend {
	return newFriend("filehelper", self)
}

// SendTextToMp 发送文本消息给公众号
func (s *Self) SendTextToMp(mp *Mp, text string) (*SentMessage, error) {
	return s.sendTextToUser(mp.User, text)
}

// SendImageToMp 发送图片消息给公众号
func (s *Self) SendImageToMp(mp *Mp, file *os.File) (*SentMessage, error) {
	return s.sendImageToUser(mp.User, file)
}

// SendFileToMp 发送文件给公众号
func (s *Self) SendFileToMp(mp *Mp, file *os.File) (*SentMessage, error) {
	return s.sendFileToUser(mp.User, file)
}

// SendVideoToMp 发送视频消息给公众号
func (s *Self) SendVideoToMp(mp *Mp, file *os.File) (*SentMessage, error) {
	return s.sendVideoToUser(mp.User, file)
}

func (s *Self) sendMessageWrapper(message *SentMessage, err error) (*SentMessage, error) {
	if err != nil {
		return nil, err
	}
	message.Self = s
	return message, nil
}
