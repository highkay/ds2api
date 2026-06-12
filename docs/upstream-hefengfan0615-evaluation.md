# hefengfan0615/ds2api 上游评估报告

评估日期：2026-05-19

评估对象：

- `CJackHwang/ds2api:main`：`8316cf8`
- `hefengfan0615/ds2api:main`：`28cad19`
- 本地当前分支：`2b4e2af`
- GitHub compare：`https://github.com/CJackHwang/ds2api/compare/main...hefengfan0615:ds2api:main`

2026-05-25 增量复查：`hefengfan0615/ds2api:main` 已更新到 `37f6206`。新增提交继续扩大 Qwen2API 改造范围，包括 Qwen 模型目录、Qwen SSE `choices[].delta` 解析、忽略 PoW、WebUI 模型加载和账号相关 `/v1/models`。这些变化进一步证明该分支已偏离本仓库的 DeepSeek Web upstream owner，不适合作为 DS2API 上游修复整体吸收。

## 结论

`hefengfan0615` 分支不适合整体合入。它混合了两类完全不同的改动：

1. 请求外形/指纹实验：随机 `device_id`、随机 DeepSeek header、随机 TLS ClientHello。
2. 协议兼容削弱：禁用工具提示词注入、禁用 OpenAI/Responses 工具调用提取、取消 `DS2API_TOOLS.txt` 上传、删除 Output integrity guard。

第一类可以保留为“需要真实账号 A/B 验证”的候选，但目前无证据显示它优于本仓库当前请求外形。第二类已经被目标包测试证明会破坏现有 XML-first tool-call 协议契约，应明确拒绝。

## 变更拆解

### 1. 请求外形/指纹

该分支在 `1371e10` 中做了这些改动：

- `Login` 的 `device_id` 从固定 `"deepseek_to_api"` 改为每次登录随机 UUID-like 字符串。
- 登录和鉴权请求从 `BaseHeaders` 改为 `GetRandomizedHeaders()`。
- header 增加或随机化 `User-Agent`、`Accept-Language`、`Accept-Encoding`、`Connection`、`Cache-Control`、`Pragma`、`DNT`、`Upgrade-Insecure-Requests`，并概率性加入 `X-Requested-With`。
- TLS ClientHello 从固定 Safari 指纹改为在 Chrome、Firefox、Safari、`HelloRandomized*` 之间随机。

这个方向的动机是规避上游把请求识别为自动化客户端，但实现方式偏粗糙：

- 每次登录随机 `device_id` 会让同一账号看起来像持续换设备，不一定比稳定设备 ID 更安全。
- TLS 指纹跨浏览器和随机模板跳变，和 Android DeepSeek app header 不一致，反而可能增加异常特征。
- `math/rand` 全局随机不可复现，失败后难以定位是哪组 header/TLS 组合触发。
- 该分支没有提供真实登录、建会话、PoW、completion 成功率数据。

本地当前分支已经有一部分更稳的请求外形改造：

- `DeviceID(accountIdentifier)` 基于账号标识稳定生成，而不是所有账号共享固定字符串，也不是每次随机。
- `BaseHeaders` 已包含 `x-rangers-id` 和 `x-client-timezone-offset`。
- transport 使用 Chrome uTLS、强制 HTTP/1.1、OkHttp-like `OkHttp-Preemptive` header，并带 cookie jar。

因此，如果要继续吸收请求指纹方向，建议从“稳定 profile”开始，而不是照搬随机化。

### 2. 工具调用链路

该分支在 `3fd5f8e`、`b038dbd` 中删除或绕过了核心工具链路：

- `promptcompat.BuildOpenAIPrompt` 不再注入工具说明，直接返回空 `toolNames`。
- OpenAI non-stream formatter 不再输出 `tool_calls`，即使模型输出了可解析工具调用。
- OpenAI chat stream 不再通过 `toolstream.ProcessChunk`/`Flush` 产生 tool call delta。
- OpenAI Responses stream 不再产生 function-call item 和 arguments events。
- `current_input_file` 模式不再上传 `DS2API_TOOLS.txt`，只上传历史正文。

这不是“优化 AI 响应”，而是把工具调用能力降级为纯文本输出。更严重的是，部分路径仍可能给出 `finish_reason=tool_calls`，但缺少对应 `tool_calls` delta 或 function-call events，形成协议不一致。

本仓库的协议契约是 XML-first：

- prompt 只把 canonical XML 作为可执行工具调用格式。
- Go 和 Node/Vercel stream 必须保持工具筛分语义一致。
- 历史 assistant `tool_calls` 会转成 XML 保留在 prompt 中。

`hefengfan0615` 没有修改 Node/Vercel streaming 实现，因此如果只合入 Go 侧禁用工具链路，还会制造本地 Go runtime 和 Vercel Node runtime 的行为漂移。

### 3. Output integrity guard

该分支在 `2a314f4` 删除了 `Output integrity guard` 系统提示注入。

该 guard 的目标是降低上游上下文、工具输出或解析残片出现乱码、重复、半截结构时被模型继续模仿的概率。删除它可能减少一点 prompt 干扰，但分支没有提供样本证明它导致了稳定回归。

如果后续确实发现 guard 在某些模型/任务中污染输出，合理方案是做 runtime/config 开关和样本回放测试，而不是直接删除默认保护。

## 动态验证

### 目标包测试

命令：

```bash
go test ./internal/promptcompat ./internal/format/openai ./internal/httpapi/openai/chat ./internal/httpapi/openai/responses ./internal/deepseek/protocol ./internal/deepseek/client ./internal/deepseek/transport
```

本地当前分支结果：通过。

`hefengfan0615` 临时 worktree 结果：失败，失败点集中在协议兼容链路：

- `internal/promptcompat`
  - 工具名为空。
  - 缺少最终 tool-call anchor instruction。
  - 缺少 Output integrity guard。
  - 缺少 read-like tool cache guard。
- `internal/format/openai`
  - schema-declared string 参数保护测试 panic，因为 expected `tool_calls` 被删除。
- `internal/httpapi/openai/chat`
  - thinking 中工具调用无法提升为 structured `tool_calls`。
  - stream finalize fallback 缺少 tool call delta。
  - Vercel prepare 缺少 DSML/XML 工具提示。
  - `current_input_file` 只上传 1 个文件，缺少 tools 文件。
- `internal/httpapi/openai/responses`
  - stream 不再发 function-call item / arguments done events。
  - thinking/hidden-thinking 工具调用无法提升。
  - schema-declared string 参数保护失效。

这组测试足以证明：禁用工具调用和删除 guard 不是可接受 backport。

### 无凭证网络探针

由于当前工作区没有 `config.json`，无法执行真实账号四段探针。为验证网络/TLS/header 层是否至少能打到 DeepSeek login endpoint，使用临时测试对三个 profile 发送无效账号登录请求，只观察 HTTP 状态和上游错误分类。

结果：

| Profile | 结果 |
| --- | --- |
| 本地当前分支，`DeepSeek/1.8.0 Android/35` + `x-rangers-id` + Chrome uTLS/OkHttp 外形 | HTTP 200，上游返回 `PASSWORD_OR_USER_NAME_IS_WRONG` |
| `CJackHwang/main`，`DeepSeek/2.0.4 Android/35` + Safari uTLS | HTTP 200，上游返回 `PASSWORD_OR_USER_NAME_IS_WRONG` |
| `hefengfan0615`，随机 UA/header/TLS，8 次无效登录 | 前 4 次 HTTP 200 + `PASSWORD_OR_USER_NAME_IS_WRONG`，后 4 次 HTTP 200 + `TOO_MANY_REQUESTS` |

这个探针只能说明三种外形都能通过网络/TLS 层到达 login endpoint；不能证明 `hefengfan0615` 的随机化提高真实账号成功率。相反，连续随机登录仍会触发上游限流，说明它不是绕过风控的充分条件。

## 证据缺口

目前未完成真实四段业务验证，原因是本地无可用 `config.json` 账号凭证。

需要补齐的探针：

1. `Login`：真实 email/mobile + password，记录 HTTP status、`code`、`biz_code`、`biz_msg`，不打印 token。
2. `CreateSession`：用登录 token 建会话，记录 session id 是否返回、错误分类和耗时。
3. `GetPow`：请求 completion target 的 PoW，记录 challenge 是否可解、错误分类和耗时。
4. `CallCompletion`：发送最小 prompt，记录 SSE 是否开始、首包耗时、最终 stop/error、403/429/空回复/静默账号分类。

对比 profile 至少应包括：

- 当前本地稳定 profile。
- 官方 `CJackHwang/main` profile。
- 候选稳定 profile：更新 app version/header，但保持 per-account stable device id。
- `hefengfan0615` 原始随机 profile，仅作为负面对照。

## 建议

1. 不合入 `hefengfan0615` 分支。
2. 不合入禁用工具调用、删除 `DS2API_TOOLS.txt`、删除 Output integrity guard 的改动。
3. 请求指纹方向只保留为实验题，不直接照搬：
   - 保留 per-account stable `device_id`。
   - 不使用跨浏览器随机 TLS。
   - 不每次请求随机 header；如果要多 profile，应按账号或进程稳定选择。
   - 用真实四段探针证明成功率提升后，再做 config/runtime 开关。
4. 如果优先级足够，下一步应写一个可复用的 live fingerprint A/B harness，而不是在业务代码里直接试错。
5. 2026-05-25 新增的 Qwen2API 改造不应纳入 DS2API 主线。账号感知模型列表可以作为独立 WebUI/接口需求重新设计，但必须保留 DeepSeek `p/v` SSE 解析、PoW、Go/Node stream 对齐和当前模型别名契约。

## 2026-06-12 跟进

本轮刷新 `hefengfan0615/ds2api:main` 到 `bfa9a2d`。该分支重新回到 DeepSeek Web 请求外形方向，同时混入默认公共代理池、区域性 GOPROXY/npm 镜像和仓库内 `deepseek.py` 参考实现。

已按本仓库边界吸收的低风险小块：

- 上游返回非 JSON body 时，`postJSONWithStatus` / `getJSONWithStatus` 不再静默返回空 map，而是返回包含 HTTP status 和 body preview 的 decode error。
- 登录失败、`biz_code` 失败和缺 token 错误会带脱敏 body preview；`token`、`password`、`authorization`、`cookie`、`secret`、`credential` 等字段会被替换为 `<redacted>`。
- 标准 completion payload 增加 DeepSeek Web 兼容的空操作字段 `action:null` 和 `preempt:false`。
- 增加 `live_deepseek_probe` build tag 下的真实四段 A/B 探针测试：登录 -> 建会话 -> PoW -> completion；同账号可跑 `current-android` 与 `candidate-web` 两种 profile。测试只记录阶段、耗时、状态和字节数，不打印 token/password/完整响应体。
- 增加同 build tag 下的多轮 ban-risk A/B 探针：同账号重复跑两种 profile，统计 `ok`、登录/session/PoW/completion 失败、403、429、空回复，以及封禁、限流、验证码、风控等风险分类。这个探针才用于判断请求外形是否可能降低封禁；单次四段探针只用于验证链路可达。

仍未采用的请求外形/运行时改动：

- 不改默认登录 profile：仍保持稳定 `DeviceID(accountID)`、`os=Android` 和现有 base headers。
- 不改默认 session payload：仍发送 `{"agent":"chat"}`，未改成空 body。
- 不采用标准 HTTP 优先登录、Web headers、随机 device id、`os=web`、mobile `area_code="+86"`。
- 不采用默认公共代理池、区域性构建镜像默认值或把 `deepseek.py` 作为运行时资产。

真实四段 A/B 已用 `admin@fnos` 上的线上实例凭据补跑。实例形态：

- 容器：`ds2api`，镜像 `ghcr.io/highkay/ds2api:sha-6d143e8`，`6011 -> 5001`。
- 配置：`/home/admin/ds2api/config.json`，`accounts=86`，`proxies=0`，`credential_slots=86`。
- 线上配置读回：`86/86` 个账号带 `muted` 标记或未来 `mute_until`。
- 最近 24 小时容器日志持续出现 `/v1/chat/completions` 的 `401` 和 `429`，说明线上账号池已经处在不可用或风险冷却状态。

探针先跑同账号 `current-android` vs `candidate-web`。早期只看 status/bytes 时，`candidate-web` 曾出现 HTTP 200 且 121 bytes，后续补强 body 分类后确认这是误判：响应体是 `{"code":0,"data":{"biz_code":5,"biz_msg":"user is muted",...}}`，不是 SSE 内容。

补强后的真实结果：

- `current-android` 在首个线上账号上登录/session/PoW 通过，但 completion 返回 `account is muted until ...`。
- `candidate-web` 对前 6 个线上账号抽样：0 个成功；5 个 completion `biz_code=5 user is muted`，1 个登录 `biz_code=10 USER_IS_BANNED`。
- `candidate-web` 对稀疏账号下标 `10/20/30/40/50/60/70/80` 抽样：0 个成功；一部分登录 `USER_IS_BANNED`，一部分登录/session/PoW 通过但 completion `user is muted`。

结论：`candidate-web` 没有在当前线上凭据池证明能减少封禁或绕过 mute。更重要的是，completion 阶段必须把 HTTP 200 JSON 错误体视作失败，不能用 status 200 证明请求外形有效。默认运行时仍保持 `current-android` 形态；`hefengfan0615` 的 Web headers、随机 Web `device_id`、`os=web`、空 session body 和标准 HTTP 优先登录不进入正式代码。

降低封禁不能用静态 diff、无效账号登录或 HTTP 200 空判断证明。需要在存在健康账号池时继续用 `TestLiveDeepSeekBanRiskABProbe` 做同账号、同代理、同模型低频 A/B；只有候选 profile 在 completion 成功率不下降的前提下显著减少 `risk_events`、403、429、JSON 风控错误或空回复，才允许进入正式运行时代码。
