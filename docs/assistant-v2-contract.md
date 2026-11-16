# AI 消息 v2：跨仓库接口契约

版本：2026-10-29（2026-11-14 补记 §1.3–§1.5、§13–§16）。依据：《VEVE AI × IM：后端与渠道调整清单》v1 及其数字员工补充（下称“清单”）。
本文只定跨仓库的接口；各仓库内部实现自行决定。字段以 `assistant/assistant.proto` 为准。

## 0. 版本与发布

- protocol 发布 tag `v0.1.97-alpha.17-new`。im-server、im-chat、im-sdk-core、im-sdk-cpp 的 `replace github.com/openimsdk/protocol => github.com/yuyuehui/protocol` 统一升到该版本。
- `SchemaVersion` 仍为 1。未发布分支的数据不兼容（方案 §11.6），升级前清理测试环境 AI 试验数据。
- 重试按钮本期不做：**重试 = 新任务新消息**，同一条 215 消息的 runID 不变。SDK 收到 runID 变化的 2122 仍然拒绝写库。

## 1. protocol 变更摘要

| 项 | 内容 |
| --- | --- |
| 过程步骤 | `ProgressBlock.steps / counts`，`ProcessStep`、`StepKind`、`StepStatus`、`ProcessCounts` |
| 进度增量 | `AssistantStreamNotification` 的 `textDelta` 与新增的 `progress`（`ProgressDelta`）放进 oneof `delta`；进度帧复用外层 `blockID`（指向 ProgressBlock），`utf8Offset` 必须为 0 |
| 交互状态 | `InteractionRef.state / outcome / actorUserID / decidedAt`，`optional` 的 `confirmerUserID`（2026-11-14）；`InteractionState`、`InteractionOutcome` |
| 云文档卡片 | `Block.docCard`：`DocCardBlock{ tid, docType, title }`（2026-11-14，§1.3） |
| 回复提醒 | `AssistantElem.mentions`：`optional Mentions{ userIDs }`（2026-11-14，§1.5） |
| 附件 IM 文件 | `AttachmentBlock` 的 `optional` `kind / imObject / width / height`，服务端写（2026-11-14，§16） |
| 未受理 | `RUN_STATUS_NOT_ACCEPTED = 10`，终态；只允许 notice 块；可不带 taskID / runID / streamEpoch，`sourceID` 与 `revision ≥ 1` 必填 |
| 函数 | `ApplyStreamFrame`（取代 `ApplyTextDelta`）、`ValidatePublicSnapshot`（新增步骤、交互状态、未受理规则）、`IsTerminal`（含 NOT_ACCEPTED）、`OutcomeAllowed`、`UnsafePublicText`、`CheckPublicProgress`、`CheckPublicProgressDelta`、`InboundAttachmentID`、错误 `ErrStreamFrame`、`ErrUnsafePublicText`；`ValidDocCard`、`MaxDocCards`、`MaxMentions`、`ValidIMObject`、`MaxIMObjectBytes`、`AttachmentKindImage / Video / Audio / File`（2026-11-14） |

### 1.1 交互状态规则

| state | outcome | actorUserID | decidedAt | 写入方 |
| --- | --- | --- | --- | --- |
| PENDING | 空 | 空 | 0 | 执行端创建交互后保存 WAITING 快照时 |
| SUBMITTED | 与类别匹配，或 CANCELLED | 提交人 | 提交时间 | 服务端：用户提交决定成功后 |
| ACCEPTED | 同上 | 同上 | 同上 | 服务端：执行端 `assistant.control.ack` 后 |
| EXPIRED | 空 | 空 | 过期时间 | 服务端 worker |
| CANCELLED | 空 | 发起停止的用户（可空） | 作废时间 | 服务端：WAITING 时停止，或运行进入终态 |

- 执行端保存的快照中，交互块必须与服务端按交互记录推导出的值**完全一致**，否则 `permission`。执行端只能原样保留服务端给出的交互块，不能自行修改状态字段。
- `confirmerUserID`（2026-11-14，proto3 `optional`）：交给发起人以外的人处理时为该人，**不设＝发起人处理**，不会输出空串；创建时定、之后不变。写操作确认（执行审批，或创建时标了 `confirm` 的问答、草稿，见 §2.6）在数字员工的工作里交给工作的确认人，群里没开自行确认的是雇佣人，这时才有值；开了自行确认的、私聊的、普通渠道机器人的都是发起人，不设。客户端只给处理人显示操作按钮（没有这个字段时为发起人）。
  - 发起人处理时不设是为了兼容（§1.4）：老客户端严格解析 215 / 2122，这样只有交给别人确认的新流程才需要新客户端。
  - 执行端保存快照时不带（老执行端）或带空串的由服务端按记录补上，重放原快照也按已保存的补；带了发起人的按不设算；带了别人、和记录不一致的按 `permission` 拒绝。
- 交互块在后续快照中保留（不移除），完成后仍显示处理结果。

### 1.2 公开文本安全

`title / detail / groupKey / summary` 进入群消息。服务端对 checkpoint、finish、append 调用 `CheckPublicProgress` / `CheckPublicProgressDelta`，命中即拒绝（reason `unsafe_public_text`）。渠道插件在发送前按同一组规则（`assistant/public_text.go` 的 `unsafePublicPatterns`）把命中的文本替换为通用文案，例如 detail 置空、title 改为“执行工具”。

### 1.3 云文档卡片（2026-11-14）

执行端在回复里放云文档卡片：`Block.docCard = DocCardBlock{ tid, docType, title }`。

- 字段规则（`ValidDocCard`，与 IM 文档卡片消息 110 一致）：`tid` 匹配 `^[A-Za-z0-9_-]{1,64}$`；`docType` 取 `doc | excel | board`；`title` 1–200 个字符，不能全是空白，不含控制字符。一条消息最多 `MaxDocCards = 10` 张。不合格的整份快照按参数错误拒绝。
- 不带链接和所有者：客户端按自己配置的文档站拼链接（`doc → /files/editor`、`excel → /files/sheet`、`board → /files/board`，`?tid=<tid>`），执行端指定不了跳转地址；打开时由文档站按点开的人核验权限。服务端不核验文档是否存在、谁能看。
- 群里发布方式为「发起人确认后发到群里」的工作：回复里有文档卡片的和有成果附件的一样，结束后整条等发起人发布，发布前别人只看到状态（卡片在群里点开就能看到文档）。
- 老客户端认不出这个块，会把整条回复显示成「无法显示的 AI 消息」（§1.4），所以由 im-chat 开关控制：`chat-api-bot.yml` 的 `assistant.docCardBlock`（环境变量 `CHATENV_CHAT_API_BOT_ASSISTANT_DOCCARDBLOCK`），默认关，能显示文档卡片的客户端普及后再打开。
  - 打开后网关在 `connected` 的 capabilities 里声明 `doc_card_block`（§6）。
  - 没打开时网关拒绝快照里有文档卡片的 checkpoint / finish（`doc_card_disabled`，§2.2），不转给 im-server。
  - 插件只在声明了 `doc_card_block` 时发文档卡片，否则把文档链接写进正文。im-server 本身总是接受合格的文档卡片。
- 被引用时：引用带文档卡片的 AI 回复，入站 `docCards` 列出这些卡片，`replyTo` 摘要里是「[文档] 标题」（§7）。

### 1.4 老客户端兼容：公开快照新增字段的规则（2026-11-14）

im-server 发 215 内容、2122 快照时按 protojson `EmitUnpopulated` 输出（普通字段是零值也输出）；已发布的 SDK 用严格的 `protojson.Unmarshal` 解析 215 / 2122，认不出的字段或 oneof 成员会让整条解析失败：215 显示为「无法显示的 AI 消息」，2122 被丢弃。所以：

- 后加进公开快照（`AssistantElem` 和其中的块）的字段必须有 presence：proto3 `optional` 或 oneof 成员。没设时不输出，老客户端不受影响，只有用到新字段的消息才需要新客户端。`confirmerUserID` 按此用 `optional`。
- 新的块类型（oneof 成员）一出现老客户端就显示不了整条，要按能力开关控制（如 §1.3 的 `doc_card_block`）。
- `assistant/validate_test.go` 的 `TestLaterSnapshotFieldsOmittedWhenUnset` 检查没设的新字段不输出，新增字段时加进去。

### 1.5 群里提醒发起人（2026-11-14）

- 群（`g_` / `sg_`）里的回复消息（215）发出时带 `MsgData.AtUserIDList = [发起人]`：服务端按群 @ 记录改发起人的会话，会话列表显示「有人@我」，老客户端也有，不需要开关。回复在受理时就建，提醒也在这时；之后的快照更新（2122）只改内容，不再提醒，结束时也不另发提醒。私聊不带，未受理提示（§5）不带。
- 快照字段 `mentions`（`optional Mentions{ repeated string userIDs }`）：群里受理时由服务端填发起人，PC 在回复开头显示「@昵称」（昵称按群成员取），SDK 可据此把这条算作 @我。执行端保存快照时带的不算，服务端一律按受理时定的写回，重放也按保存的比对。规则：1–`MaxMentions = 10` 个不重复的用户 ID，每个 1–64 字节。
- 老客户端认不出 `mentions`，整条会显示不了（§1.4），所以由 im-server 开关控制：`openim-api.yml` 的 `assistant.replyMentions`（环境变量 `IMENV_OPENIM_API_ASSISTANT_REPLYMENTIONS`），默认关，各端 SDK（PC、移动端、Web）都认这个字段后再开。开关只影响打开以后受理的回复。

## 2. im-server 内部接口（网关 → im-server，`/assistant/internal/*`）

### 2.1 错误返回

失败时保留现有 HTTP 状态码，响应体统一为：

```json
{ "errCode": <int>, "errMsg": "<reason>", "data": { "reason": "<reason>", "retryAfterMs": 0 } }
```

`reason` 为稳定的机器码：

| reason | 场景 | 执行端处理 |
| --- | --- | --- |
| `rate_limited` | append 超过每个 run 10 帧/秒 | 合并后在 `retryAfterMs` 之后重发，**不换 streamSeq** |
| `reply_not_ready` | 回复尚未 stored 时发流帧 | 等 `assistant.reply.query` 显示 stored 后再发 |
| `reply_revoked` | 回复已撤回 | 中止运行，不再写入 |
| `run_terminal` | 运行已终态 | 停止写入 |
| `revision_conflict` | expectedRevision / baseRevision 过时（含服务端写入了 STOPPING、交互状态） | 调 `assistant.snapshot` 对齐后重试 |
| `stream_gap` | 序号缺口 | 调 `assistant.snapshot` 对齐后从 `lastStreamSeq + 1` 继续 |
| `payload_too_large` | 快照或正文超限 | 改为输出附件并加 NoticeBlock（§2.4） |
| `unsafe_public_text` | 公开过程含路径、URL、密钥等 | 按 §1.2 替换后重发 |
| `permission` / `arguments` / `not_found` / `unavailable` | 其他 | 按原逻辑处理 |

### 2.2 网关透传

im-chat 网关不再把失败收成笼统的一句话：上游返回 `data.reason` 时，ack 为

```json
{ "ok": false, "errCode": 1006, "errMsg": "<reason>", "result": { "reason": "<reason>", "retryAfterMs": 0 } }
```

`rate_limited` 使用现有 `ErrRateLimited = 1003`；其余 assistant 失败使用新增的 `ErrAssistant = 1006`。网络错误或上游无法解析时仍返回 1001 和原提示（结果未知，先查询原动作）。

网关自己拒绝、不调 im-server 的：`doc_card_disabled`（2026-11-14）——im-chat 没打开 `assistant.docCardBlock` 时，checkpoint / finish 的快照里有文档卡片（§1.3），ack 为 `errCode: 1006`、reason `doc_card_disabled`。执行端去掉文档卡片、把链接写进正文后重发（im-server 没收到过，同一 actionID 可以重用）。

### 2.3 限频

只在 im-server 做一处：`assistant.append` 按 `(taskID, runID)` 使用 Redis 令牌桶，速率 10 帧/秒、容量 10。网关对 assistant 动作不另设限频。

### 2.4 体积上限

| 位置 | checkpoint / finish | 其他 assistant 动作 |
| --- | --- | --- |
| im-chat 网关单帧（`gateway/assistant.go`）及 WebSocket 读上限 | 1 MiB | 256 KiB |
| im-server `assistantBind` | 1 MiB + 64 KiB（网关会注入 actionID 等字段，留余量） | 256 KiB + 64 KiB |
| im-server live（Redis）快照 | 1 MiB（不变） | — |

单条回复全部 TextBlock 的正文合计不超过 **256 KiB（UTF-8）**，超出返回 `payload_too_large`，禁止静默截断。执行端在接近上限时停止追加正文，把完整内容作为输出附件，并加 `NoticeBlock{code: "content_truncated_to_attachment"}`。

### 2.5 流式（append）

请求不变：`{ "frame": <AssistantStreamNotification 的 protojson> }`，`textDelta` 与 `progress` 二选一。服务端：

- 用 `ApplyStreamFrame` 合并进 Redis live；进度帧先过 `CheckPublicProgressDelta`。
- 回复撤回或运行终态时，直接返回 `reply_revoked` / `run_terminal`。
- 授权结果按 `(connectionID, taskID, streamEpoch)` 缓存 5–10 秒；checkpoint / finish 仍做完整校验；撤回、退群、换连接时失效。

### 2.6 控制读取与确认

- `assistant.control.read`：`{ "interactionID": "..." }` 行为不变；新增 `{ "taskID": "..." }` 读取任务级控制，返回 `{ "pending": true }` 或 `{ "pending": false, "kind": "stop", "taskID", "runID", "actionID", "requestedAt", "reason"? }`（`reason` 同 §4）。
- 停止的确认方式是执行端 `assistant.finish`（status = STOPPED，expectedRevision 为 STOPPING 快照的 revision）。
- 写操作确认（2026-11-14）：`assistant.interaction.create` 新增 `confirm`（bool，问答、草稿可标）。标了的和执行审批一样是写操作确认：数字员工的工作里交给工作的确认人处理（`InteractionRef.confirmerUserID`），创建时生成确认令牌。返回体新增 `confirmerUserID`（空串＝发起人处理）和 `durable`（创建时没传 `expiresAt` 的不设有效期的交互，§14）。执行端把 `confirmerUserID` 原样放进交互块即可，空串和不带一样，服务端保存时去掉。
  - 群里的工作发起写操作确认（执行审批，或 `confirm: true` 的问答、草稿）要发起人档位到可写，否则 `tier_insufficient`；没标 `confirm` 的草稿照旧不算写操作。
  - 草稿交给的确认人不是发起人、又带附件时，附件要已发布到会话（确认人读得到），否则 `permission`。
  - 同一 actionID 重试改了 `confirm` 按 `conflict` 拒绝。补充材料等要发起人本人回答的问答不要标 `confirm`。
- 交互的决定（`assistant.control.read` 带 `interactionID`）新增 `decidedBy`（做决定的人）、`confirm`（是写操作确认时为 true）。写操作确认被同意（approve、answer、confirm）时另带：
  - `confirmToken`：确认令牌，服务端签发，同一交互不变、重拉不变。执行端凭它执行写操作，可作写接口的幂等键（重放返回首次结果）；不交给模型，不写进公开内容。
  - `paramHash`：令牌绑定的内容摘要。执行审批为 `execution.fingerprint`；草稿为冻结草稿的摘要；问答为问题和答案的摘要。
  - 拒绝、取消和没标 `confirm` 的不带；上线前创建的交互没有令牌，不补。
- 问答选项（2026-11-14）：`assistant.interaction.create` 新增 `questions`（选项题），决定新增 `answers`（按题回答），见 §15。
- `assistant.control.ack`（交互）成功后，服务端把交互块改为 ACCEPTED 并发 2122。**ack 返回体带最新快照 `snapshot`**，执行端用它替换本地快照，再继续保存。

### 2.7 快照

`assistant.snapshot` 返回 live 合并后的快照（含 Redis 中的正文与步骤，以及 `lastStreamSeq`）。

## 3. im-server 用户接口（客户端 → im-server）

| 路径 | 请求 | 说明 |
| --- | --- | --- |
| `POST /assistant/stop` | `{ taskID, runID, expectedRevision, actionID }` | 仅发起人。ACCEPTED / RUNNING / WAITING → STOPPING，写快照并发 2122；WAITING 时当前交互置 CANCELLED。返回 `{ snapshot }`。重复 actionID 幂等 |
| `POST /assistant/snapshot` | `{ conversationID, serverMsgID }` 或 `{ taskID }` | 返回 `{ snapshot, lastStreamSeq }`，运行中为 live 合并内容。发起人及能看到该消息的当前会话成员可读 |
| `POST /assistant/sources` | 现有入参 | 会话成员可读公开来源（§3.9）；私有来源仍只给发起人 |
| `POST /assistant/control` | 现有入参；问答带选项题时可交 `answers`（§15） | 提交决定成功后，服务端写 SUBMITTED 并发 2122 |

停止超时：STOPPING 后 30 秒内执行端未 `finish(STOPPED)`，worker 写 UNKNOWN 并发 2122，不报“已停止”。

切换执行设备：旧连接的运行在心跳租约（30 秒）过期后由服务端写 UNKNOWN 并发 2122；绑定旧运行的交互一律拒绝提交，并置 EXPIRED；旧连接迟到的写入按连接 fencing token 拒绝。新设备不接续、不重放。

## 4. 控制下行推送（im-server → im-chat → 插件）

im-server 与 im-chat 使用同一个 Redis 库。

- im-server 在用户提交决定、请求停止后，以 JSON 写入：
  - `RPUSH bot:assistant:control:pending:<botID> <event>`（键中不带花括号），随后 `LTRIM -100 -1`、`EXPIRE 600`；
  - `PUBLISH bot:assistant:control <event>`。
- event：`{ "eventID", "botID", "taskID", "runID", "interactionID"?, "kind", "actionID", "reason"?, "ts" }`；`kind` 取 `stop | answer | approve | reject | confirm | cancel`。**不带答案、参数或草稿正文。**
- `reason` 只在 stop 上有，说明为什么停：没有＝发起人自己停的（`/assistant/stop` 或会话里的停止命令）；`source_revoked`＝触发消息撤回了；数字员工的工作结束了时是工作的结束原因（agent `Work.endReason`）：`stopped`（发起人或雇佣人停止了工作）、`employee_disabled`（员工停用）、`not_eligible`（雇佣人失去使用资格）、`tier_changed`（发起人在群里的档位不够了）、`instance_removed`（实例被移出群）、`group_dismissed`（群解散）、`group_use_disabled`（关了群使用）、`unhired`（移除了雇佣）、`initiator_left`（发起人退群）、`parent_ended`（协作子工作所属的工作结束了）、`revoked`（需求消息撤回了，补充材料的任务）等。老版本 agent 服务写的停止没有细分原因，是 `work_stopped` / `work_aborted`。执行端一律按停止处理（`finish(STOPPED)`），`reason` 只用于给用户说明，认不出的值当一般停止。
- im-chat 网关订阅该频道，找到 botID 的本地连接后下发 `DownEvent{ event: "assistant.control", eventId: eventID, ts, data: { taskID, runID, interactionID, kind, actionID, reason } }`（`interactionID`、`reason` 没有时不带）。新连接在 `connected` 之后补发 pending 列表中的全部事件。
- 插件按 eventId 去重，收到后调 `assistant.control.read` 读取详情。兜底轮询改为 5 秒。

## 5. 未受理提示消息（im-chat → im-server）

- im-chat 在 `dispatchReceive` 发现 bot 不在线时调用 `POST /assistant/internal/system/notice`（断言只认 `/assistant/internal/` 前缀），携带 IM 管理员 token 与 `X-Assistant-Gateway-Assertion`（`GatewayClaims`：`ConnectionID = "system"`、`ActionID = eventID`）。
- 请求：`{ conversationID, sourceID, initiatorID, code }`，`code` 取 `bot_offline`（普通渠道机器人）或 `device_offline`（数字员工）。
- im-server 校验：来源消息存在于该会话且由 initiator 发送，bot 是单聊对方或被 @；以 bot 身份发送 215：`AssistantElem{ status: NOT_ACCEPTED, sourceID, revision: 1, blocks: [notice{ code, summary }] }`。
  - clientMsgID = digest(`["assistant-notice", botID, sourceID]`)，重复请求幂等；
  - 同一 `(botID, conversationID)` 60 秒内只发一条；
  - summary 取 im-server 配置 `api.assistant.notices.<code>.zh`，客户端可按 code 本地化（配置同时提供 `en`）；
  - 不创建任务，上线后不补做。
- 数字员工：所属人的私聊和群 @ 回复 `device_offline`；非所属人回复简短的 `bot_unavailable`（“暂不可用”，不透露执行设备状态，独立的幂等键与 60 秒去重窗口）（im-chat 判断，§12 修订）。
- 返回 `{ serverMsgID }` 或 `{ deduplicated: true }`。

## 6. 网关握手 v2（插件 ↔ im-chat）

- 插件请求头：`X-Bot-Protocol-Version: 2`、`X-Bot-Capabilities: assistant,progress,control.push,inbound.v2`；数字员工另加 `X-Bot-Device-ID`、`X-Bot-Host-Epoch`。服务端同时接受 1 和 2，v1 行为不变。
- 数字员工连接：与这个员工的执行租约比对 deviceID、hostEpoch，一致才放行：先看按员工的租约 `bot:de:host:{ownerUserID}:<botID>`（新插件 `host/sync` 拿的），没有再看整人租约 `bot:de:host:{ownerUserID}`（老插件 `host/acquire`）。缺少请求头或不一致：HTTP **409**，响应头 `X-Bot-Reject-Reason: host_replaced`，正文 `host_replaced`。Redis 读取失败：503。普通渠道机器人不受影响。连接建立后，每次上行动作同样复核租约；不一致时 `closeWith("host_replaced")`。
- `hostEpoch`（执行租约代次；按员工的和整人的共用一个计数，不会撞号）与 `streamEpoch`（流）、连接 fencing token 分开命名，不复用。
- `connected` 事件 data 新增 `{ protocolVersion: 2, capabilities: [...] }`；装配了 assistant 后端时含 `assistant`、`progress`、`control.push`、`inbound.v2`、`work`（§13），im-chat 打开了 `assistant.docCardBlock` 时另含 `doc_card_block`（§1.3）。
- 插件：`host_replaced`（409 或 disconnect reason）→ 状态 `replaced`，不重连；401/403 → `auth_failed`，不重连；其他 → 1s→60s 退避重连。`assistantProtocol` 默认开启，仅在服务端声明 `assistant` 能力时生效。

## 7. 入站 payload v2（仅协议 2 的连接）

`message.receive` 的 `data.message` 新增：

```json
{
  "sourceID": "<触发消息 serverMsgID，网关已核验>",
  "mentions": ["<AtUserIDList>"],
  "replyTo": { "serverMsgID": "", "seq": 0, "contentType": 0, "senderID": "" },
  "attachments": [{ "attachmentID": "att_…", "sourceMsgID": "", "type": "image|file|video|audio", "name": "", "size": 0, "mimeType": "" }],
  "docCards": [{ "sourceMsgID": "", "tid": "", "docType": "doc|excel|board", "title": "" }]
}
```

- v2 附件**不带 URL**。序号细则：内容缺失的 item 也占序号（但不下发）；Voice 只在单附件消息中计为 0，混合消息中不计；带 items 的 Quote 消息自身 items 中的附件按 MixedTextPicture 规则、以引用消息自身的 serverMsgID 计。
- `attachmentID = InboundAttachmentID(sourceMsgID, index)`，index 为该消息中附件按出现顺序的序号：Picture / File / Video / Voice 消息为 0；MixedTextPicture 按 items 中 picture / file / video 的顺序；引用消息的附件使用被引用消息的 serverMsgID 与其自身序号。
- im-server 在 `assistant.open` 时按同一规则，把触发消息与明确引用消息中的附件登记为任务输入资源，资源 ID 即 attachmentID。需要校验发起人与 bot 都可见，上限最多 10 个、单个图片 6 MiB 其他（文件、视频、语音）16 MiB、合计 50 MiB（2026-11-14 和桌面端统一成一套，原为 5 个 / 单个 20 MiB），按出现顺序逐个判断：先看单个大小，再看个数，再看合计（按消息里声明的大小，下载时按实际大小复核）；下载由服务端完成。插件通过 `inputs.list / read / chunk` 读取。
- 无引用时 `replyTo` 省略；`mentions` 直接取消息的 `AtUserIDList`。
- `docCards`（2026-11-14）：文档卡片（自定义消息 110，`type: "doc_card"`）私聊直接发给 bot，或在群里引用这张卡片并 @ bot 时下发；引用带文档卡片（DocCardBlock）的 AI 回复时列出其中的卡片（按出现顺序，最多 10 张）；没有卡片时省略。`sourceMsgID` 是卡片所在的消息（直接发来的是触发消息，引用的是被引用的那条）。`message.text` 为 `[文档] 标题`（协议 1 只有它）。链接和所有者不下发。卡片按 im-server 发送时的规则严格校验（`pkg/strictsend`：外层和卡片都不能有多余字段），不合格的和其他 110 一样按不支持处理（§12 `unsupported_message`）。读文档内容要以发起人身份走文档服务，代理凭证还没定。
- 引用 AI 回复（215，2026-11-14）：其中的成果附件（AttachmentBlock）按出现顺序计序号（以这条 215 的 serverMsgID 计）列入 `attachments`，`replyTo` 摘要取公开正文和文档卡片标题。im-server 在 open 时把它们登记为入站资源，字节取自原工作的输出资源（不经 URL、不另存副本），每次读取都复验原工作仍把它公开在本会话。引用的引用（Quote）仍只作来源、不展开附件。
- 雇佣实例引用入群前的消息（2026-11-14）：实例看不到入群前的群消息，这类引用按雇佣人核验，登记附件、读取、写入公开来源都看雇佣人能不能看；雇佣人删了消息或退群后不能再读。触发消息不放宽。

## 8. 托管模式宿主步骤（PC 宿主 → 插件）

宿主 watch 快照在 `progress` 字符串之外新增 `steps`，元素与 `ProcessStep` 的 JSON 一致（camelCase，`kind` / `status` 用枚举名字符串），另加可选 `counts`。插件只按 §1.2 过滤后转发，不自行推断。主仓库侧按此字段实现。

## 9. SDK（im-sdk-core）

- 2121：文本帧与进度帧都透传，回调不变（`OnRecvCustomBusinessMessage`，key `assistant.stream`）。
- 2122：校验被拒时仍发 `assistant.snapshot` 业务事件，附 `rejectReason`（`stale_revision | terminal | run_changed | revoked | local_missing | invalid`）；按 eventID 去重。
- NOT_ACCEPTED 的提示消息正常入库。
- 校验不通过的 215 不丢弃，保留原内容入库，由前端显示“无法显示的 AI 消息”。

## 10. 资源类接口补充（im-server 实现后补记）

- **入站附件读取**：`assistant.inputs.list` 返回 `kind: "inbound"` 的批次，`resources[].resourceID` 即 attachmentID（带 name / mediaType / size / type / sourceMsgID，下载前 sha256 为空）；超限或不可用的写入 `rejected[]`（`unavailable | too_large | too_many | total_too_large`），open 不因此失败。返回体另带 `inboundLimits { maxCount, maxImageBytes, maxFileBytes, maxTotalBytes }`（2026-11-14，§7 的上限），执行端告诉用户上限时照它说，不另写一套数字。读取用 `assistant.inputs.chunk { taskID, batchID, resourceID, offset }`，首次读取可能等待服务端下载；`inputs.read` 对二进制返回 `skipped[]`（`binary_use_chunk`）。
- **输出附件直接公开**：`assistant.output.shared { taskID, resourceID, version, sha256 }` 由原执行连接声明，只放行本运行生成、已 ready、目标为当前会话的附件；返回 `{ shareID, conversationID, state: "shared", attachment }`，`attachment` 原样作为 AttachmentBlock 写入快照。不带 resourceID 时保持原列表行为。
- **公开来源**：SourceRef 要求 `conversationID` 为当前会话、`sourceID == serverMsgID`、`seq ≥ 1`、`locator` 为空、`contentVersion` 为消息 content 的 sha256 hex，最多 20 条且不重复。`/assistant/sources` 对发起人返回 `audience: private`，对其他成员只返回快照中的公开来源并按其可见范围过滤（`audience: public`），附件只给名称与类型。SourceRef 没有附件名称字段，消息内需要显示附件名时须另加协议字段。
- **上下文分页**：`assistant.context.read` 新增 `limit`（1–100）、`cursor`（上一页返回值）、`since`（毫秒，24 小时内）；返回项含 `contentType` 与 `attachments[{ attachmentID, type, name, mimeType, size }]`，不含 URL。文档卡片（2026-11-14，自定义消息 110）返回 `text: "[文档] 标题"` 和 `docCard { tid, docType, title }`（规则同 §7 `docCards`，不含链接和所有者），其他自定义消息仍计入 `excluded`。
- **通用审批**：`execution = { hostApprovalID, hostRunID, operation, parameters, title?, description? }`，operation 匹配 `^[a-z0-9_.:-]{1,64}$`；`schedule.create` 保留严格参数校验，其他操作的 parameters 为 JSON 对象（≤32 KiB）；title（≤200 字）/ description（≤2000 字）只出现在受保护详情与执行端决定中，并绑定进指纹。

## 11. 第二轮修正（2026-10-29）

- **公开文本规则**：`public_text.go` 已补全：中文后紧跟的路径、`/data/` 等更多前缀与 `./`、`../` 相对路径、全角冒号、中文密钥关键词、不带协议的域名、IPv4、JWT。新增两项按代码判断的规则，插件须逐条移植：token 的值长度 ≥ 8 且含字母才算密钥（“Token: 10”不拦）；不含 `_ - .` 的 32 位以上字母数字串，数字占比 ≥ 1/8，或长度 ≥ 40 且含 `+`/`/`，才算密钥或编码数据（长文件名不拦）。
- **执行设备代次**：im-chat 在 `bot:conn:<botID>` 的连接状态中写入 `hostEpoch`（非数字员工为 0）与 `deviceID`。im-server 在 `assistant.open` 时记录当时连接的 hostEpoch（有按员工的租约时是这个员工那一份的，只在这个员工换电脑时变）；`assistant.resume` 若当前连接的 hostEpoch 与记录不同，拒绝（reason `host_changed`），旧运行由租约过期转 UNKNOWN。同一设备断线重连（hostEpoch 不变）仍可 resume。
- **运行租约**：im-server 在 open / resume 时登记运行租约，heartbeat、append、checkpoint 时续期；30 秒无任何活动转 UNKNOWN。插件在整个运行期间每 10 秒发送一次 `assistant.runtime.heartbeat`。
- **执行端转交停止请求**：新增动作 `assistant.stop.request { taskID, runID, sourceID }`（im-chat 路由到 `/assistant/internal/stop/request`）。sourceID 为用户在会话中发出的停止命令消息；im-server 校验该消息在同一会话、发送者为发起人，然后走与 `/assistant/stop` 相同的流程（STOPPING + 2122 + 控制推送），返回 `{ snapshot }`。插件收到 `/stop` 文本命令时调用它，不再直接写 STOPPED。
- **交互块不可删除**：快照中出现过的交互块在后续快照中必须保留（状态由服务端推导），缺失返回 `permission`。
- **握手能力约束**：协议 2 的握手必须带 `X-Bot-Capabilities` 且包含 `inbound.v2`，否则 400 并返回 `X-Supported-Capabilities`。未声明 `assistant` 的连接不能调用 `assistant.*` 动作；未声明 `control.push` 的连接不下发 `assistant.control`。
- **控制推送时序**：连接在 `connected` 事件入队之前不接收实时控制推送；之后从 pending 列表补发，保证 `connected` 先于任何 `assistant.control` 到达。
- **来源块**：新协议下每次运行都在 open 后读取 `assistant.sources.read`，至少把触发消息本身作为公开来源写入 SourcesBlock（普通群 @ 提问也生成），`counts.messages` 与之一致。
- **插件不再退回文本**：渠道插件去掉 `streaming`、`assistantProtocol` 两个开关（配置字段保留但被忽略，兼容旧配置），回复一律使用 215，握手固定为协议 2。服务端未声明 `assistant` 能力时插件不回复也不运行模型，只记录错误；不再有新消息的文本通道。客户端渲染历史文本消息的能力保留。OpenClaw outbound 适配器（`message.send`，agent 主动发往其他会话）不受影响。

## 12. 第三轮体验修正（2026-10-29）

- **非文本消息下发**：im-chat 不再静默丢弃合并转发、名片、位置、表情、代码、Markdown、高级文本等类型，`message.text` 放一句摘要（例如“[合并转发] 群聊记录”“[位置] 地址”“[名片] 昵称”），原始内容类型保留在 `contentType`。确实无法支持的类型，im-chat 调 `/assistant/internal/system/notice`，code 为 `unsupported_message`（im-server 配置 `notices.unsupported_message.zh/en`），适用范围与离线提示一致（单聊或被 @；数字员工只回复所属人）。
- **引用失败保留原话**：引用核验失败时保留用户正文，失败说明作为附注，不再整段替换。
- **可引用 AI 回复与引用消息**：引用目标为 215 时，`replyTo.contentType = 215`，引用摘要取其公开正文（TextBlock）；引用目标为 Quote 时取其自身文字。im-server 的来源核验、公开来源与入站附件登记对这两类引用同样接受（被引用的 Quote 不登记附件，被引用的 215 登记其中的成果附件（2026-11-14，§7），触发消息自身附件照常登记），不再整条拒绝。
- **群名**：群消息的 `groupName` 按群资料填充。
- **触发消息撤回即停止**：触发消息被撤回时，im-server 自动把未结束的运行置为 STOPPING（停止人记为系统，交互作废），发 2122 并推送 `assistant.control` stop（`reason: source_revoked`）；执行端照常 `finish(STOPPED)`，超时按 UNKNOWN。
- **数字员工离线时回复非所属人**：取代 §5 中“其他人不额外回复”。非所属人的消息回复 `bot_unavailable`（im-server 配置 `notices.bot_unavailable.zh/en`）；`unsupported_message` 仍只回复所属人。

## 13. 数字员工的工作与 message.send（2026-11-14 补记）

雇佣实例的消息按工作派发（agent 服务的 `Work`，`/agent/work/get` 可查）。只有同时满足这几条的连接按工作派发：雇佣实例、协议 2、`X-Bot-Capabilities` 声明了 `work`、网关装配了 assistant 后端（`connected` 的 capabilities 含 `work`）。执行电脑在线但连接没声明 `work` 的照老办法直接下发，不建工作；不在线的建工作排队，上线后补派。

- `message.receive` 的 `data.work`（只在按工作派发的消息上有）：

  ```json
  { "workID": "", "token": "", "supplement": false, "taskID": "", "tier": 4, "publishMode": 1, "selfConfirm": false, "confirmerID": "" }
  ```

  - `token`：只在新派的工作上带。同一条消息再派一次会换新令牌，插件按 serverMsgID 去重、用最新的。这份工作的 `assistant.*` 动作都在信封 `workToken` 里带回（网关只把摘要转给 im-server），补充材料沿用所属工作的令牌。im-server 默认不核对（`openim-api.yml` 的 `assistant.requireWorkToken` 关，环境变量 `IMENV_OPENIM_API_ASSISTANT_REQUIREWORKTOKEN`），插件都带上以后再打开。
  - `supplement`：这条是所属工作的补充材料；`taskID`：补充到的工作已经开始时的 215 任务。
  - `tier`、`publishMode`、`selfConfirm`、`confirmerID`：这份工作的设置，字段名和取值与 `Work` 相同（`GroupTier`、`GroupPublishMode`），取建工作时定下的；补充材料带所属工作的，离线排队、上线补派的也带。`tier` 2 仅对话、3 只读、4 可写（不到可写不能发写操作确认，见 §2.6）；`publishMode` 1 发起人确认后发到群里、2 直接发；`confirmerID` 是写操作确认的处理人（群里没开 `selfConfirm` 的是雇佣人，其余是发起人）；交互的 `confirmerUserID`（§1.1）在处理人就是发起人时不设，其余和它一致。值为 0 / false / 空的省略，私聊一般只有 `confirmerID`。
- 上行 `message.send`（OpenClaw outbound，主动发往别的会话）：
  - `markdown: true` 或 `contentType: 118` 时按 Markdown（118，正文 `{"content": "..."}`）发，其余一律按文本发。im-server 不收 118 时（参数错误，发送前就拒）网关自动退回文本，ack 照常成功。
  - `attachments`（2026-11-14）：只收 `file.upload` 领到的 `uploadID`，网关不下载外部链接。im-chat `chat-api-bot.yml` 配了 `gateway.objectURLPrefix` 才打开，这时 `connected` 的 capabilities 含 `message_attachments`；没打开时 `file.upload` 和带附件的 `message.send` 都返回失败 ack（`errCode: 1001`，`errMsg: "attachment send not enabled"`）。协议 1 的连接也能用。
    1. 上行 `file.upload`，`data { type, name, size, mimeType }`：`type` 为 `image|file|video|audio`，video、audio 按文件发；图片只收 png、jpeg、gif、webp、bmp，最大 20 MiB，别的图片按 `file` 发；其余最大 100 MiB；`name` 去掉路径后不超过 255 字节，不能带控制字符和改变文字方向的字符。成功 ack 的 `result { uploadID, url, file, header, formData, expires, successCodes }`。
    2. 以 multipart/form-data POST 到 `url`：带上 `header`，`formData` 原样，文件放在 `file` 指定的字段里、排最后；返回码在 `successCodes` 里才算成功，`expires`（毫秒时间戳，约 15 分钟）前传完。
    3. 上行 `message.send`：`content` 留空，`attachments: [{ "uploadID": "", "width": 0, "height": 0 }]`，一条只带一个附件、不带正文；图片必须带宽高（1–100000）。图片按 102、其余按 105 发出。第一次发时核对文件已传上去、大小和 `size` 一致；`uploadID` 1 小时内有效、只有领它的机器人能用，可以发到多个会话。
    - `file.upload` 每个机器人每秒 2 次，超了 `errCode: 1003`；同一 `actionId` 5 分钟内重发回放上次的 ack（失败也一样），重试换新的 `actionId`。各 `errMsg` 的含义和处理见 im-chat `docs/digital-employee-client-api.md` §4.13“发图片、文件”。
  - 雇佣实例在群里不能用 `message.send`，返回失败 ack（`errCode: 1001`）：群里的档位和发布方式（如确认后发布）只对 215 生效，要用 215 回复。私聊和普通渠道机器人不受影响。

## 14. 不设有效期的交互与执行电脑离线（2026-11-14）

数字员工的工作要等人处理时，执行电脑可能关机。原来租约 30 秒过期就写 UNKNOWN，要你处理的交互随之作废；改为：

- **创建**：`assistant.interaction.create` 不传 `expiresAt` 即为不设有效期的交互（传了的照旧不超过 24 小时）。有效期由服务端定为 15 天（工作 14 天没人处理就结束，多留 1 天让第 14 天做的决定还有 24 小时等电脑上线）。返回体带 `durable: true` 和算出的 `expiresAt`。同一 actionID 重试改了是否设有效期按 `conflict` 拒绝。
- **离线也能处理**：这类交互不绑定创建它的连接，执行电脑不在线时也能提交决定（其余交互照旧要原连接在线，否则 `unavailable`）；决定由接续这次运行的连接读取、受理。
- **挂起**：执行租约过期时，运行在 WAITING、且没处理完的交互全是不设有效期的，不写 UNKNOWN，改为挂起（快照内容不变，revision 加 1，照常发 2122）：
  - 还有没处理的交互：挂到它们到期为止；
  - 都处理完了：挂到最晚的决定后 24 小时为止（不超过交互到期）。
  - 到时电脑还没回来写 UNKNOWN。有设了有效期的未决交互，或运行不在 WAITING 的，照旧写 UNKNOWN。
- **接续**：同一台电脑回来后照常 `assistant.resume`（revision 已变，先 `assistant.snapshot` 对齐；`revision_conflict` 同样处理）。挂起的运行掉线较久、hostEpoch 已变，按登记的 `X-Bot-Device-ID` 认同一台电脑，设备 ID 不同仍是 `host_changed`；原连接没断的，心跳或 resume 直接解除挂起。接续后服务端把掉线期间已处理、还没受理的交互再推一次 `assistant.control`，执行端照常 `control.read` 读决定、`control.ack` 受理。
- **停止**：挂起中的运行没人确认停止，直接写 STOPPED（不经 STOPPING）；电脑回来接续时看到已停止。
- **agent 工作（`Work`）**：
  - `hostOffline`：执行电脑不在线、开始了的任务挂起等电脑回来，工作没结束时才有；`agent.work.changed` 推送同样带。
  - `waitTime`：这次进入「要你处理」的时间（中间有人处理过的从最近一次处理算），只在要你处理时有。电脑不在线的时间也算。
  - 要你处理满 3 天，员工在原会话里提醒一次：私聊发给发起人（文本），群里 @ 要处理的人（快照里还没处理的交互的确认人，没写的是发起人；AtText）。消息 `ex` 为 `{"agentReminder":"waiting","workID":"…","sourceMsgID":"…"}`，客户端据此认出提醒、按 workID 找到工作卡片。同一次等待只发一条。
  - 满 14 天仍未处理：工作结束为已停止，`endReason: wait_expired`，任务按停止处理（§4 的 `reason`）；处理完了只等电脑上线超过 24 小时：未完成 `expired`。
  - 后台扫描（要开 agent 的 `jobs.enabled`）用 agent_work 集合的新索引 `{status: 1, wait_time: 1}`，服务启动时自动建。

## 15. 问答选项（2026-11-14）

问答（QUESTION）可以带选项题：PC 显示成选项按钮，一次可以问几道题。选项和回答都不进公开快照，交互块的 summary 仍是固定文案。

- **创建**：`assistant.interaction.create` 新增 `questions`，只有问答能带；`question` 仍必填，当标题（只有一道题时可以和题目一样，服务端拼文字时不重复）。

  ```json
  { "questions": [ { "id": "format", "text": "用什么格式？", "options": [ { "id": "pdf", "label": "PDF", "description": "适合打印" }, { "id": "word", "label": "Word" } ], "multiple": false, "allowCustom": true } ] }
  ```

  - 1–10 道题；每题 0–10 个选项，没选项＝只填文字；`multiple` 可多选（至少 2 个选项）；`allowCustom` 有选项时也能自己填（没选项的题不用标）。
  - 题和选项的 `id` 1–64 个字符，只能是字母、数字和 `_` `.` `-`，题号不重复、题内选项号不重复；`text` 1–500 字，`label` 1–100 字、题内不重复，`description` 可省、最多 200 字；`text`、`label` 不能是空白。
  - 不合规按 `arguments` 拒绝；同一 actionID 重试改了题目按 `conflict` 拒绝。
- **详情**（`POST /assistant/interaction/detail`）：带选项题时多 `questionTitle`（执行端给的标题）和 `questions`（同上）；`question` 换成服务端拼的文字（标题、题目、选项字母和说明、「可多选」「也可以自己填」和回复提示），老 PC 照样能看、能写文字回答。没选项题的问答不带这两个字段，`question` 不变。
- **回答**（`POST /assistant/control`）：新增 `answers`，和 `answer` 二选一，`decision` 为 `answer`：

  ```json
  { "interactionID": "", "actionID": "", "decision": "answer", "expectedRevision": 1, "answers": [ { "id": "format", "options": ["pdf"], "text": "" } ] }
  ```

  - 每道题恰好答一次；`options` 是这道题的选项号、不重复，单选最多一个；`text` 是自己填的，只有没选项或 `allowCustom` 的题收，最多 4096 字节、不能是空白；每题至少选一个或填了字。不合规按 `arguments` 拒绝。
  - 服务端按题目顺序（题内按选项顺序）整理，换个顺序重交算同一次提交。
  - 老 PC 只交 `answer` 文字，照样收。新 PC 等 im-server 上线后再交 `answers`：老服务端按未知字段拒绝。
- **决定**（执行端 `assistant.control.read`）：按题回答的多 `answers`（整理过的，格式同上）；`answer` 是服务端拼好的文字（每题一行题目、一行「答：选项文字、…；补充：自己填的」，多道题带题号），只认 `answer` 的执行端照样能用。只交文字的没有 `answers`，插件两种都要能处理。写操作确认的 `paramHash` 在按题回答时连同 `answers` 算，只交文字的和以前一样。

## 16. 成果附件复制成 IM 文件（2026-11-14）

原来成果附件只能按 `resourceID` 走 `/assistant/resource/read` 下载，输出资源准备 15 分钟后就读不了，也不能转发。im-server 开关 `openim-api.yml` 的 `assistant.attachmentIMFile`（环境变量 `IMENV_OPENIM_API_ASSISTANT_ATTACHMENTIMFILE`）打开后，成果按普通 IM 文件处理：

- **复制**：成果第一次出现在快照的附件块里时（checkpoint / finish，各项核对都通过后），服务端把冻结的成果复制成一份普通 IM 文件（同一 bucket 的 `openim/data/assistant/` 下，登记到 IM 对象表，上传人记为执行端机器人），附件块带上：
  - `kind`：`image | video | audio | file`，按真实字节判断。能解出尺寸的 JPEG / PNG / GIF / WebP（边长 ≤ 30000、像素 ≤ 64M）才算图片；视频、音频要声明的 mediaType 和内容嗅探对得上；其余都是 file。
  - `imObject`：IM 对象名 `<机器人ID>/msg_assistant_<32 位十六进制><扩展名>`（机器人 ID 带了要转义的字符时开头换成 `assistant`）。规则 `ValidIMObject`：按 `/` 分段，每段只有字母、数字和 `. _ @ -`，最长 `MaxIMObjectBytes = 1024` 字节。
  - `width`、`height`：只有图片带，都大于 0。
  - 四个字段都是 `optional`（§1.4）；`kind` 和 `imObject` 要么都有要么都没有，不合规的整份快照按参数错误拒绝。
- **客户端用法**：下载拼 `<API 地址>/object/<imObject>`（和 IM 图片、文件消息一样，对象名直接拼、不再转义）；图片缩略图加 `?type=image&width=<宽>&height=<高>`。下载得到的文件名是对象名，保存时用附件块的 `name`。转发按普通 IM 图片 / 文件消息发；视频没有封面和时长，转成视频消息要客户端自己补，或按文件发。没带这几个字段的附件块照旧按 `resourceID` 读。
- **内容类型**：复制出的对象用服务端判断的 Content-Type：图片按真实格式；会被浏览器当页面或脚本执行的（`text/html`、`image/svg+xml`、`*+xml`、`*/javascript` 等，或内容嗅探是 HTML / XML 的）一律 `application/octet-stream`。
- **权限**：复制出的是普通 IM 文件，和 IM 上传的文件一样拿到对象名就能下载（转发出去的人也能），不再按任务核验，到期和 IM 上传的文件一样处理。群里「发起人确认后发到群里」的工作发布前不对别人输出附件块（原有规则），对象名不会提前露出。
- **只在第一次出现时复制**：资源记录上记下复制结果（`im_file`），重试、并发写入都用同一份；之后的写入沿用上一版快照里的字段。执行端写的这几个字段一律不算：没开开关时清掉，开了按服务端记录写回；重放按已保存的快照比对。复制失败（10 秒超时、存储出错等）不影响写入，只记日志，这个附件块不带字段，之后也不补。
- **不覆盖**：只管写进快照的附件块，私发（deliveries）不复制；不提供单独的复制接口。
- **15 分钟期限**：输出资源准备 15 分钟后就读不了。原来长任务、等了很久的交互之后再写快照时，已在快照里的附件会被重新核对而失败；现在上一版快照里已有的同一份成果（资源、版本、摘要、名称、类型、大小都一样）不再核对，与开关无关；新出现的过期成果照旧拒绝。没开开关时 PC 仍只能在 15 分钟内按 `resourceID` 下载。
- **开关**：默认关。老客户端认不出这几个字段，整条回复会显示不了（§1.4），各端 SDK（PC、移动端、Web）都认这几个字段后再开。要同时开 `resourcesEnabled`；OSS 的 RAM 要允许读 `assistant/resources/ready/`、写 `openim/data/assistant/`，复制时设了目标对象 ACL（跟随 bucket），还要 `oss:PutObjectAcl`。关掉开关只影响之后第一次出现的附件，已保存的快照里的字段不撤回。
