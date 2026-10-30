# AI 消息 v2：跨仓库接口契约

版本：2026-10-29。依据：《VEVE AI × IM：后端与渠道调整清单》v1 及其数字员工补充（下称“清单”）。
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
| 交互状态 | `InteractionRef.state / outcome / actorUserID / decidedAt`；`InteractionState`、`InteractionOutcome` |
| 未受理 | `RUN_STATUS_NOT_ACCEPTED = 10`，终态；只允许 notice 块；可不带 taskID / runID / streamEpoch，`sourceID` 与 `revision ≥ 1` 必填 |
| 函数 | `ApplyStreamFrame`（取代 `ApplyTextDelta`）、`ValidatePublicSnapshot`（新增步骤、交互状态、未受理规则）、`IsTerminal`（含 NOT_ACCEPTED）、`OutcomeAllowed`、`UnsafePublicText`、`CheckPublicProgress`、`CheckPublicProgressDelta`、`InboundAttachmentID`、错误 `ErrStreamFrame`、`ErrUnsafePublicText` |

### 1.1 交互状态规则

| state | outcome | actorUserID | decidedAt | 写入方 |
| --- | --- | --- | --- | --- |
| PENDING | 空 | 空 | 0 | 执行端创建交互后保存 WAITING 快照时 |
| SUBMITTED | 与类别匹配，或 CANCELLED | 提交人 | 提交时间 | 服务端：用户提交决定成功后 |
| ACCEPTED | 同上 | 同上 | 同上 | 服务端：执行端 `assistant.control.ack` 后 |
| EXPIRED | 空 | 空 | 过期时间 | 服务端 worker |
| CANCELLED | 空 | 发起停止的用户（可空） | 作废时间 | 服务端：WAITING 时停止，或运行进入终态 |

- 执行端保存的快照中，交互块必须与服务端按交互记录推导出的值**完全一致**，否则 `permission`。执行端只能原样保留服务端给出的交互块，不能自行修改状态字段。
- 交互块在后续快照中保留（不移除），完成后仍显示处理结果。

### 1.2 公开文本安全

`title / detail / groupKey / summary` 进入群消息。服务端对 checkpoint、finish、append 调用 `CheckPublicProgress` / `CheckPublicProgressDelta`，命中即拒绝（reason `unsafe_public_text`）。渠道插件在发送前按同一组规则（`assistant/public_text.go` 的 `unsafePublicPatterns`）把命中的文本替换为通用文案，例如 detail 置空、title 改为“执行工具”。

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

- `assistant.control.read`：`{ "interactionID": "..." }` 行为不变；新增 `{ "taskID": "..." }` 读取任务级控制，返回 `{ "pending": true }` 或 `{ "pending": false, "kind": "stop", "taskID", "runID", "actionID", "requestedAt" }`。
- 停止的确认方式是执行端 `assistant.finish`（status = STOPPED，expectedRevision 为 STOPPING 快照的 revision）。
- `assistant.control.ack`（交互）成功后，服务端把交互块改为 ACCEPTED 并发 2122。**ack 返回体带最新快照 `snapshot`**，执行端用它替换本地快照，再继续保存。

### 2.7 快照

`assistant.snapshot` 返回 live 合并后的快照（含 Redis 中的正文与步骤，以及 `lastStreamSeq`）。

## 3. im-server 用户接口（客户端 → im-server）

| 路径 | 请求 | 说明 |
| --- | --- | --- |
| `POST /assistant/stop` | `{ taskID, runID, expectedRevision, actionID }` | 仅发起人。ACCEPTED / RUNNING / WAITING → STOPPING，写快照并发 2122；WAITING 时当前交互置 CANCELLED。返回 `{ snapshot }`。重复 actionID 幂等 |
| `POST /assistant/snapshot` | `{ conversationID, serverMsgID }` 或 `{ taskID }` | 返回 `{ snapshot, lastStreamSeq }`，运行中为 live 合并内容。发起人及能看到该消息的当前会话成员可读 |
| `POST /assistant/sources` | 现有入参 | 会话成员可读公开来源（§3.9）；私有来源仍只给发起人 |
| `POST /assistant/control` | 现有入参 | 提交决定成功后，服务端写 SUBMITTED 并发 2122 |

停止超时：STOPPING 后 30 秒内执行端未 `finish(STOPPED)`，worker 写 UNKNOWN 并发 2122，不报“已停止”。

切换执行设备：旧连接的运行在心跳租约（30 秒）过期后由服务端写 UNKNOWN 并发 2122；绑定旧运行的交互一律拒绝提交，并置 EXPIRED；旧连接迟到的写入按连接 fencing token 拒绝。新设备不接续、不重放。

## 4. 控制下行推送（im-server → im-chat → 插件）

im-server 与 im-chat 使用同一个 Redis 库。

- im-server 在用户提交决定、请求停止后，以 JSON 写入：
  - `RPUSH bot:assistant:control:pending:<botID> <event>`（键中不带花括号），随后 `LTRIM -100 -1`、`EXPIRE 600`；
  - `PUBLISH bot:assistant:control <event>`。
- event：`{ "eventID", "botID", "taskID", "runID", "interactionID"?, "kind", "actionID", "ts" }`；`kind` 取 `stop | answer | approve | reject | confirm | cancel`。**不带答案、参数或草稿正文。**
- im-chat 网关订阅该频道，找到 botID 的本地连接后下发 `DownEvent{ event: "assistant.control", eventId: eventID, ts, data: { taskID, runID, interactionID, kind, actionID } }`。新连接在 `connected` 之后补发 pending 列表中的全部事件。
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
- 数字员工连接：与 Redis 租约 `bot:de:host:{ownerUserID}` 的 deviceID、hostEpoch 比对，一致才放行。缺少请求头或不一致：HTTP **409**，响应头 `X-Bot-Reject-Reason: host_replaced`，正文 `host_replaced`。Redis 读取失败：503。普通渠道机器人不受影响。连接建立后，每次上行动作同样复核租约；不一致时 `closeWith("host_replaced")`。
- `hostEpoch`（账号执行租约）与 `streamEpoch`（流）、连接 fencing token 分开命名，不复用。
- `connected` 事件 data 新增 `{ protocolVersion: 2, capabilities: [...] }`；装配了 assistant 后端时含 `assistant`、`progress`、`control.push`、`inbound.v2`。
- 插件：`host_replaced`（409 或 disconnect reason）→ 状态 `replaced`，不重连；401/403 → `auth_failed`，不重连；其他 → 1s→60s 退避重连。`assistantProtocol` 默认开启，仅在服务端声明 `assistant` 能力时生效。

## 7. 入站 payload v2（仅协议 2 的连接）

`message.receive` 的 `data.message` 新增：

```json
{
  "sourceID": "<触发消息 serverMsgID，网关已核验>",
  "mentions": ["<AtUserIDList>"],
  "replyTo": { "serverMsgID": "", "seq": 0, "contentType": 0, "senderID": "" },
  "attachments": [{ "attachmentID": "att_…", "sourceMsgID": "", "type": "image|file|video|audio", "name": "", "size": 0, "mimeType": "" }]
}
```

- v2 附件**不带 URL**。序号细则：内容缺失的 item 也占序号（但不下发）；Voice 只在单附件消息中计为 0，混合消息中不计；带 items 的 Quote 消息自身 items 中的附件按 MixedTextPicture 规则、以引用消息自身的 serverMsgID 计。
- `attachmentID = InboundAttachmentID(sourceMsgID, index)`，index 为该消息中附件按出现顺序的序号：Picture / File / Video / Voice 消息为 0；MixedTextPicture 按 items 中 picture / file / video 的顺序；引用消息的附件使用被引用消息的 serverMsgID 与其自身序号。
- im-server 在 `assistant.open` 时按同一规则，把触发消息与明确引用消息中的附件登记为任务输入资源，资源 ID 即 attachmentID。需要校验发起人与 bot 都可见，上限 5 个 / 单个 20 MiB / 合计 50 MiB，下载由服务端完成。插件通过 `inputs.list / read / chunk` 读取。
- 无引用时 `replyTo` 省略；`mentions` 直接取消息的 `AtUserIDList`。

## 8. 托管模式宿主步骤（PC 宿主 → 插件）

宿主 watch 快照在 `progress` 字符串之外新增 `steps`，元素与 `ProcessStep` 的 JSON 一致（camelCase，`kind` / `status` 用枚举名字符串），另加可选 `counts`。插件只按 §1.2 过滤后转发，不自行推断。主仓库侧按此字段实现。

## 9. SDK（im-sdk-core）

- 2121：文本帧与进度帧都透传，回调不变（`OnRecvCustomBusinessMessage`，key `assistant.stream`）。
- 2122：校验被拒时仍发 `assistant.snapshot` 业务事件，附 `rejectReason`（`stale_revision | terminal | run_changed | revoked | local_missing | invalid`）；按 eventID 去重。
- NOT_ACCEPTED 的提示消息正常入库。
- 校验不通过的 215 不丢弃，保留原内容入库，由前端显示“无法显示的 AI 消息”。

## 10. 资源类接口补充（im-server 实现后补记）

- **入站附件读取**：`assistant.inputs.list` 返回 `kind: "inbound"` 的批次，`resources[].resourceID` 即 attachmentID（带 name / mediaType / size / type / sourceMsgID，下载前 sha256 为空）；超限或不可用的写入 `rejected[]`（`unavailable | too_large | too_many | total_too_large`），open 不因此失败。读取用 `assistant.inputs.chunk { taskID, batchID, resourceID, offset }`，首次读取可能等待服务端下载；`inputs.read` 对二进制返回 `skipped[]`（`binary_use_chunk`）。
- **输出附件直接公开**：`assistant.output.shared { taskID, resourceID, version, sha256 }` 由原执行连接声明，只放行本运行生成、已 ready、目标为当前会话的附件；返回 `{ shareID, conversationID, state: "shared", attachment }`，`attachment` 原样作为 AttachmentBlock 写入快照。不带 resourceID 时保持原列表行为。
- **公开来源**：SourceRef 要求 `conversationID` 为当前会话、`sourceID == serverMsgID`、`seq ≥ 1`、`locator` 为空、`contentVersion` 为消息 content 的 sha256 hex，最多 20 条且不重复。`/assistant/sources` 对发起人返回 `audience: private`，对其他成员只返回快照中的公开来源并按其可见范围过滤（`audience: public`），附件只给名称与类型。SourceRef 没有附件名称字段，消息内需要显示附件名时须另加协议字段。
- **上下文分页**：`assistant.context.read` 新增 `limit`（1–100）、`cursor`（上一页返回值）、`since`（毫秒，24 小时内）；返回项含 `contentType` 与 `attachments[{ attachmentID, type, name, mimeType, size }]`，不含 URL。
- **通用审批**：`execution = { hostApprovalID, hostRunID, operation, parameters, title?, description? }`，operation 匹配 `^[a-z0-9_.:-]{1,64}$`；`schedule.create` 保留严格参数校验，其他操作的 parameters 为 JSON 对象（≤32 KiB）；title（≤200 字）/ description（≤2000 字）只出现在受保护详情与执行端决定中，并绑定进指纹。

## 11. 第二轮修正（2026-10-29）

- **公开文本规则**：`public_text.go` 已补全：中文后紧跟的路径、`/data/` 等更多前缀与 `./`、`../` 相对路径、全角冒号、中文密钥关键词、不带协议的域名、IPv4、JWT。新增两项按代码判断的规则，插件须逐条移植：token 的值长度 ≥ 8 且含字母才算密钥（“Token: 10”不拦）；不含 `_ - .` 的 32 位以上字母数字串，数字占比 ≥ 1/8，或长度 ≥ 40 且含 `+`/`/`，才算密钥或编码数据（长文件名不拦）。
- **执行设备代次**：im-chat 在 `bot:conn:<botID>` 的连接状态中写入 `hostEpoch`（非数字员工为 0）与 `deviceID`。im-server 在 `assistant.open` 时记录当时连接的 hostEpoch；`assistant.resume` 若当前连接的 hostEpoch 与记录不同，拒绝（reason `host_changed`），旧运行由租约过期转 UNKNOWN。同一设备断线重连（hostEpoch 不变）仍可 resume。
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
- **可引用 AI 回复与引用消息**：引用目标为 215 时，`replyTo.contentType = 215`，引用摘要取其公开正文（TextBlock）；引用目标为 Quote 时取其自身文字。im-server 的来源核验、公开来源与入站附件登记对这两类引用同样接受（被引用的 215 / Quote 不登记附件，触发消息自身附件照常登记），不再整条拒绝。
- **群名**：群消息的 `groupName` 按群资料填充。
- **触发消息撤回即停止**：触发消息被撤回时，im-server 自动把未结束的运行置为 STOPPING（停止人记为系统，交互作废），发 2122 并推送 `assistant.control` stop；执行端照常 `finish(STOPPED)`，超时按 UNKNOWN。
- **数字员工离线时回复非所属人**：取代 §5 中“其他人不额外回复”。非所属人的消息回复 `bot_unavailable`（im-server 配置 `notices.bot_unavailable.zh/en`）；`unsupported_message` 仍只回复所属人。
