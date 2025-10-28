// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package constant

const (

	///ContentType - 消息内容类型定义
	//UserRelated - 用户相关消息类型
	ContentTypeBegin = 100
	Text             = 101 // 文本消息
	Picture          = 102 // 图片消息
	Voice            = 103 // 语音消息
	Video            = 104 // 视频消息
	File             = 105 // 文件消息
	AtText           = 106 // @消息
	Merger           = 107 // 合并转发消息
	Card             = 108 // 名片消息
	Location         = 109 // 位置消息
	Custom           = 110 // 自定义消息
	Revoke           = 111 // 撤回消息
	Typing           = 113 // 正在输入状态
	Quote            = 114 // 引用消息
	Emoji            = 115 // 表情包消息

	AdvancedText                 = 117 // 高级文本消息
	MarkdownText                 = 118 // Markdown格式文本
	CustomNotTriggerConversation = 119 // 自定义消息(不触发会话更新)
	CustomOnlineOnly             = 120 // 自定义消息(仅在线推送)
	ReactionMessageModifier      = 121 // 消息反应修改器
	ReactionMessageDeleter       = 122 // 消息反应删除器
	Common                       = 200 // 通用消息
	GroupMsg                     = 201 // 群组消息
	SignalMsg                    = 202 // 信令消息
	CustomNotification           = 203 // 自定义通知

	// SysRelated - 系统相关通知类型
	NotificationBegin = 1000

	// 好友相关通知
	FriendApplicationApprovedNotification = 1201 // 好友申请通过通知
	FriendApplicationRejectedNotification = 1202 // 好友申请拒绝通知
	FriendApplicationNotification         = 1203 // 好友申请通知
	FriendAddedNotification               = 1204 // 好友添加通知
	FriendDeletedNotification             = 1205 // 好友删除通知
	FriendRemarkSetNotification           = 1206 // 好友备注设置通知
	BlackAddedNotification                = 1207 // 拉黑通知
	BlackDeletedNotification              = 1208 // 取消拉黑通知
	FriendInfoUpdatedNotification         = 1209 // 好友信息更新通知
	FriendsInfoUpdateNotification         = 1210 // 好友信息批量更新通知

	// 会话相关通知
	ConversationChangeNotification = 1300 // 会话设置变更通知

	// 用户相关通知
	UserNotificationBegin         = 1301
	UserInfoUpdatedNotification   = 1303 // 用户信息更新通知
	UserStatusChangeNotification  = 1304 // 用户状态变更通知
	UserCommandAddNotification    = 1305 // 用户命令添加通知
	UserCommandDeleteNotification = 1306 // 用户命令删除通知
	UserCommandUpdateNotification = 1307 // 用户命令更新通知

	UserSubscribeOnlineStatusNotification = 1308 // 用户在线状态订阅通知

	UserNotificationEnd = 1399
	OANotification      = 1400 // OA通知

	// 群组相关通知
	GroupNotificationBegin = 1500

	GroupCreatedNotification                 = 1501 // 群组创建通知
	GroupInfoSetNotification                 = 1502 // 群组信息设置通知
	JoinGroupApplicationNotification         = 1503 // 加入群组申请通知
	MemberQuitNotification                   = 1504 // 群成员退出通知
	GroupApplicationAcceptedNotification     = 1505 // 群组申请通过通知
	GroupApplicationRejectedNotification     = 1506 // 群组申请拒绝通知
	GroupOwnerTransferredNotification        = 1507 // 群主转让通知
	MemberKickedNotification                 = 1508 // 群成员被踢通知
	MemberInvitedNotification                = 1509 // 群成员邀请通知
	MemberEnterNotification                  = 1510 // 群成员加入通知
	GroupDismissedNotification               = 1511 // 群组解散通知
	GroupMemberMutedNotification             = 1512 // 群成员禁言通知
	GroupMemberCancelMutedNotification       = 1513 // 群成员取消禁言通知
	GroupMutedNotification                   = 1514 // 群组禁言通知
	GroupCancelMutedNotification             = 1515 // 群组取消禁言通知
	GroupMemberInfoSetNotification           = 1516 // 群成员信息设置通知
	GroupMemberSetToAdminNotification        = 1517 // 群成员设为管理员通知
	GroupMemberSetToOrdinaryUserNotification = 1518 // 群成员设为普通用户通知
	GroupInfoSetAnnouncementNotification     = 1519 // 群组公告设置通知
	GroupInfoSetNameNotification             = 1520 // 群组名称设置通知

	// 信令通知 (已注释，暂未使用)
	//SignalingNotificationBegin = 1600
	//SignalingNotification      = 1601
	//SignalingNotificationEnd   = 1649

	// 超级群组相关通知
	SuperGroupNotificationBegin  = 1650
	SuperGroupUpdateNotification = 1651 // 超级群组更新通知
	MsgDeleteNotification        = 1652 // 消息删除通知
	SuperGroupNotificationEnd    = 1699

	// 会话相关通知
	ConversationPrivateChatNotification = 1701 // 会话私聊通知
	ConversationUnreadNotification      = 1702 // 会话未读通知
	ClearConversationNotification       = 1703 // 清空会话通知
	ConversationDeleteNotification      = 1704 // 删除会话通知

	// 业务通知
	BusinessNotificationBegin = 2000
	BusinessNotification      = 2001 // 业务通知
	BusinessNotificationEnd   = 2099

	// 消息相关通知
	MsgRevokeNotification  = 2101 // 消息撤回通知
	DeleteMsgsNotification = 2102 // 删除消息通知
	LikeMsgNotification    = 2103 // 点赞消息通知
	HasReadReceipt         = 2200 // 已读回执

	NotificationEnd = 5000 // 通知类型结束标记

	// 消息状态
	MsgNormal  = 1 // 正常消息
	MsgDeleted = 4 // 已删除消息

	// 消息来源类型
	UserMsgType = 100 // 用户消息
	SysMsgType  = 200 // 系统消息

	// 会话类型
	SingleChatType = 1 // 单聊
	// WriteGroupChatType Not enabled temporarily
	WriteGroupChatType   = 2 // 群聊(写权限)
	ReadGroupChatType    = 3 // 群聊(读权限)
	NotificationChatType = 4 // 通知会话
	// Token状态
	NormalToken  = 0 // 正常Token
	InValidToken = 1 // 无效Token
	KickedToken  = 2 // 被踢Token
	ExpiredToken = 3 // 过期Token

	// 多终端登录策略
	DefalutNotKick = 0 // 默认不踢人
	// Full-end login, but the same end is mutually exclusive.
	AllLoginButSameTermKick = 1 // 全端登录，但同端互斥
	// The PC side is mutually exclusive, and the mobile side is mutually exclusive, but the web side can be online at
	// the same time.
	AllLoginButSameClassKick = 4 // PC端互斥，移动端互斥，但Web端可同时在线
	// The PC terminal can be online at the same time,but other terminal only one of the endpoints can login.
	PCAndOther = 5 // PC端可同时在线，其他终端只能一个端点登录

	// 在线状态
	Online  = 1 // 在线
	Offline = 0 // 离线

	// 注册状态
	Registered   = 1 // 已注册
	UnRegistered = 0 // 未注册

	// 消息接收选项
	ReceiveMessage          = 0 // 接收消息
	NotReceiveMessage       = 1 // 不接收消息
	ReceiveNotNotifyMessage = 2 // 接收消息但不通知

	// 消息选项键
	IsHistory                  = "history"                  // 是否存储历史
	IsPersistent               = "persistent"               // 是否持久化
	IsOfflinePush              = "offlinePush"              // 是否离线推送
	IsUnreadCount              = "unreadCount"              // 是否计入未读数
	IsConversationUpdate       = "conversationUpdate"       // 是否更新会话
	IsSenderSync               = "senderSync"               // 是否发送者同步
	IsNotPrivate               = "notPrivate"               // 是否非私聊
	IsSenderConversationUpdate = "senderConversationUpdate" // 是否发送者会话更新
	IsSenderNotificationPush   = "senderNotificationPush"   // 是否发送者通知推送
	IsReactionFromCache        = "reactionFromCache"        // 是否从缓存获取反应
	IsNotNotification          = "isNotNotification"        // 是否非通知
	IsSendMsg                  = "isSendMsg"                // 是否发送消息

	// 群组状态
	GroupOk              = 0 // 群组正常
	GroupBanChat         = 1 // 群组禁言
	GroupStatusDismissed = 2 // 群组已解散
	GroupStatusMuted     = 3 // 群组静音

	// 群组类型
	NormalGroup  = 0 // 普通群组
	SuperGroup   = 1 // 超级群组
	WorkingGroup = 2 // 工作群组

	GroupBaned          = 3 // 群组被封禁
	GroupBanPrivateChat = 4 // 群组禁止私聊

	// 用户加入群组来源
	JoinByAdmin = 1 // 管理员邀请

	JoinByInvitation = 2 // 邀请加入
	JoinBySearch     = 3 // 搜索加入
	JoinByQRCode     = 4 // 扫码加入

	// 存储服务配置
	MinioDurationTimes = 3600 // Minio存储时长(秒)
	AwsDurationTimes   = 3600 // AWS存储时长(秒)

	// 回调命令
	CallbackBeforeSendSingleMsgCommand                   = "callbackBeforeSendSingleMsgCommand"                   // 发送单聊消息前回调
	CallbackAfterSendSingleMsgCommand                    = "callbackAfterSendSingleMsgCommand"                    // 发送单聊消息后回调
	CallbackBeforeSendGroupMsgCommand                    = "callbackBeforeSendGroupMsgCommand"                    // 发送群聊消息前回调
	CallbackAfterSendGroupMsgCommand                     = "callbackAfterSendGroupMsgCommand"                     // 发送群聊消息后回调
	CallbackMsgModifyCommand                             = "callbackMsgModifyCommand"                             // 消息修改回调
	CallbackUserOnlineCommand                            = "callbackUserOnlineCommand"                            // 用户上线回调
	CallbackUserOfflineCommand                           = "callbackUserOfflineCommand"                           // 用户下线回调
	CallbackUserKickOffCommand                           = "callbackUserKickOffCommand"                           // 用户被踢回调
	CallbackOfflinePushCommand                           = "callbackOfflinePushCommand"                           // 离线推送回调
	CallbackOnlinePushCommand                            = "callbackOnlinePushCommand"                            // 在线推送回调
	CallbackSuperGroupOnlinePushCommand                  = "callbackSuperGroupOnlinePushCommand"                  // 超级群组在线推送回调
	CallbackBeforeAddFriendCommand                       = "callbackBeforeAddFriendCommand"                       // 添加好友前回调
	CallbackBeforeUpdateUserInfoCommand                  = "callbackBeforeUpdateUserInfoCommand"                  // 更新用户信息前回调
	CallbackBeforeCreateGroupCommand                     = "callbackBeforeCreateGroupCommand"                     // 创建群组前回调
	CallbackBeforeMemberJoinGroupCommand                 = "callbackBeforeMemberJoinGroupCommand"                 // 成员加入群组前回调
	CallbackBeforeSetGroupMemberInfoCommand              = "CallbackBeforeSetGroupMemberInfoCommand"              // 设置群成员信息前回调
	CallbackBeforeSetMessageReactionExtensionCommand     = "callbackBeforeSetMessageReactionExtensionCommand"     // 设置消息反应扩展前回调
	CallbackBeforeDeleteMessageReactionExtensionsCommand = "callbackBeforeDeleteMessageReactionExtensionsCommand" // 删除消息反应扩展前回调
	CallbackGetMessageListReactionExtensionsCommand      = "callbackGetMessageListReactionExtensionsCommand"      // 获取消息列表反应扩展回调
	CallbackAddMessageListReactionExtensionsCommand      = "callbackAddMessageListReactionExtensionsCommand"      // 添加消息列表反应扩展回调

	// 回调动作码
	ActionAllow     = 0 // 允许
	ActionForbidden = 1 // 禁止
	// 回调处理码
	CallbackHandleSuccess = 0 // 回调处理成功
	CallbackHandleFailed  = 1 // 回调处理失败

	// 文件上传类型
	OtherType = 1 // 其他类型
	VideoType = 2 // 视频类型
	ImageType = 3 // 图片类型

	// 消息发送状态
	MsgStatusNotExist = 0 // 消息不存在
	MsgIsSending      = 1 // 消息发送中
	MsgSendSuccessed  = 2 // 消息发送成功
	MsgSendFailed     = 3 // 消息发送失败
)

const (
	WriteDiffusion = 0 // 写扩散
	ReadDiffusion  = 1 // 读扩散
)

const (
	UnreliableNotification    = 1 // 不可靠通知
	ReliableNotificationNoMsg = 2 // 可靠通知(无消息)
	ReliableNotificationMsg   = 3 // 可靠通知(有消息)
)

const (
	AtAllString       = "AtAllTag" // @所有人标签
	AtNormal          = 0          // 普通@
	AtMe              = 1          // @我
	AtAll             = 2          // @所有人
	AtAllAtMe         = 3          // @所有人且@我
	GroupNotification = 4          // 群组通知
)

// 内容类型到推送内容的映射
var ContentType2PushContent = map[int64]string{
	Picture:   "[PICTURE]",      // 图片
	Voice:     "[VOICE]",        // 语音
	Video:     "[VIDEO]",        // 视频
	File:      "[File]",         // 文件
	Text:      "[TEXT]",         // 文本
	AtText:    "[@TEXT]",        // @文本
	Emoji:     "[EMOJI]",        // 表情包
	GroupMsg:  "[GROUPMSG]]",    // 群组消息
	Common:    "[NEWMSG]",       // 通用消息
	SignalMsg: "[SIGNALINVITE]", // 信令邀请
}

const (
	FieldRecvMsgOpt    = 1  // 接收消息选项字段
	FieldIsPinned      = 2  // 是否置顶字段
	FieldAttachedInfo  = 3  // 附加信息字段
	FieldIsPrivateChat = 4  // 是否私聊字段
	FieldGroupAtType   = 5  // 群组@类型字段
	FieldEx            = 7  // 扩展字段
	FieldUnread        = 8  // 未读字段
	FieldBurnDuration  = 9  // 阅后即焚时长字段
	FieldHasReadSeq    = 10 // 已读序列号字段
)

const (
	// 用户权限级别
	IMOrdinaryUser       = 0 // IM普通用户
	AppOrdinaryUsers     = 1 // 应用普通用户
	AppAdmin             = 2 // 应用管理员
	AppNotificationAdmin = 3 // 应用通知管理员
	AppRobotAdmin        = 4 // 应用机器人管理员

	// 群组权限级别
	GroupOwner         = 100 // 群主
	GroupAdmin         = 60  // 群管理员
	GroupOrdinaryUsers = 20  // 群普通用户

	// 群组响应状态
	GroupResponseAgree  = 1  // 同意
	GroupResponseRefuse = -1 // 拒绝

	// 好友响应状态
	FriendResponseNotHandle = 0  // 未处理
	FriendResponseAgree     = 1  // 同意
	FriendResponseRefuse    = -1 // 拒绝

	// 性别
	Male   = 1 // 男性
	Female = 2 // 女性
)

const (
	OperationID     = "operationID"  // 操作ID
	OpUserID        = "opUserID"     // 操作用户ID
	ConnID          = "connID"       // 连接ID
	OpUserPlatform  = "platform"     // 操作用户平台
	Token           = "token"        // Token
	RpcCustomHeader = "customHeader" // RPC中间件自定义ctx参数
	CheckKey        = "CheckKey"     // 检查键
	TriggerID       = "triggerID"    // 触发器ID
	RemoteAddr      = "remoteAddr"   // 远程地址
)

const (
	BecomeFriendByImport = 1 // 管理员导入好友
	BecomeFriendByApply  = 2 // 申请添加好友
)

const (
	ApplyNeedVerificationInviteDirectly = 0 // 申请需要同意，邀请直接进
	AllNeedVerification                 = 1 // 所有人进群需要验证，除了群主管理员邀请进群
	Directly                            = 2 // 直接进群
)

const (
	GroupRPCRecvSize = 30 // 群组RPC接收大小
	GroupRPCSendSize = 30 // 群组RPC发送大小
)

const FriendAcceptTip = "You have successfully become friends, so start chatting" // 好友接受提示

// GroupIsBanChat 检查群组是否被禁言
func GroupIsBanChat(status int32) bool {
	return status == GroupStatusMuted
}

// GroupIsBanPrivateChat 检查群组是否禁止私聊
func GroupIsBanPrivateChat(status int32) bool {
	return status == GroupBanPrivateChat
}

const LogFileName = "OpenIM.log" // 日志文件名

const LocalHost = "0.0.0.0" // 本地主机地址

// 命令行参数标志
const (
	FlagPort                  = "port"                  // 端口标志
	FlagWsPort                = "ws_port"               // WebSocket端口标志
	FlagTransferProgressIndex = "transferProgressIndex" // 传输进度索引标志
	FlagPrometheusPort        = "prometheus_port"       // Prometheus端口标志
	FlagConf                  = "config_folder_path"    // 配置文件路径标志
)

const OpenIMCommonConfigKey = "OpenIMServerConfig" // OpenIM通用配置键

const CallbackCommand = "command" // 回调命令

const BatchNum = 100 // 批处理数量

// 用户订阅常量
const (
	SubscriberUser = 1 // 订阅用户
	Unsubscribe    = 2 // 取消订阅
)

const (
	GroupSearchPositionHead = 1 // 群组搜索位置：头部
	GroupSearchPositionAny  = 2 // 群组搜索位置：任意位置
)

const (
	FirstPageNumber   = 1   // 第一页页码
	MaxSyncPullNumber = 500 // 最大同步拉取数量
)

const (
	MsgStatusSending     = 1 // 消息状态：发送中
	MsgStatusSendSuccess = 2 // 消息状态：发送成功
	MsgStatusSendFailed  = 3 // 消息状态：发送失败
	MsgStatusHasDeleted  = 4 // 消息状态：已删除
	MsgStatusFiltered    = 5 // 消息状态：已过滤
)
