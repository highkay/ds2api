# 上游 Fork 监控清单

本文记录需要定期跟踪的 DS2API 上游和活跃 fork，用于复查可吸收的实现、规避重复调研，并给出当前处理建议。

最近复查时间：2026-06-12。

## 本地引用

所有监控分支统一拉到 `refs/remotes/review/*`，避免污染正式 remote。

```bash
git fetch https://github.com/CJackHwang/ds2api +refs/heads/main:refs/remotes/review/cjack-main
git fetch https://github.com/qingdeng888/ds2api +refs/heads/main:refs/remotes/review/qingdeng888-main
git fetch https://github.com/fuwei99/ds2api +refs/heads/main:refs/remotes/review/fuwei99-main
git fetch https://github.com/ricardosantis/ds2api +refs/heads/main:refs/remotes/review/ricardosantis-main
git fetch https://github.com/tempppw01/ds2api +refs/heads/main:refs/remotes/review/tempppw01-main
git fetch https://github.com/TonyWu2333/ds2api +refs/heads/main:refs/remotes/review/TonyWu2333-main
git fetch https://github.com/emptysuns/ds2api +refs/heads/main:refs/remotes/review/emptysuns-main
git fetch https://github.com/hefengfan0615/ds2api +refs/heads/main:refs/remotes/review/hefengfan0615-main
git fetch https://github.com/whatnano/ds2api +refs/heads/main:refs/remotes/review/whatnano-main
git fetch https://github.com/voktoylo/ds2api +refs/heads/main:refs/remotes/review/voktoylo-main
git fetch https://github.com/xiaotian2333/ds2api +refs/heads/main:refs/remotes/review/xiaotian2333-main
git fetch https://github.com/tangsong404/whale2api +refs/heads/main:refs/remotes/review/tangsong404-whale2api-main
git fetch https://github.com/namlevia/ds2api +refs/heads/main:refs/remotes/review/namlevia-main +refs/heads/dev:refs/remotes/review/namlevia-dev +refs/heads/vi-localization:refs/remotes/review/namlevia-vi-localization
git fetch https://github.com/Fly143/deepseek-free-api +refs/heads/main:refs/remotes/review/fly143-main
```

已下线或不可刷新仓库只保留本地旧快照，不再放入常规 fetch 清单：

- `1cyberlangke1/dsp:main` -> `refs/remotes/review/1cyberlangke1-dsp-main`：2026-05-25 与 2026-05-27 两次确认 GitHub 返回 `repository not found`，本地 `0759d7f` 仅作历史对照。

对比新增提交：

```bash
git log --oneline <old-sha>..refs/remotes/review/<name>
git diff --stat refs/remotes/review/cjack-main..refs/remotes/review/<name>
```

## 监控对象

| 上游 | 本地 ref | 当前 HEAD | 功能主题 | 当前处理建议 |
| --- | --- | --- | --- | --- |
| `CJackHwang/ds2api:main` | `review/cjack-main` | `8316cf8` | 官方基线 | 作为 fork diff 的比较基准。 |
| `qingdeng888/ds2api:main` | `review/qingdeng888-main` | `42bc1ed` | 账号健康权重、冷却、P2C 选择、Docker config 挂载修正 | 核心账号健康能力已吸收；后续只看是否有新的调度/冷却实测修正。 |
| `fuwei99/ds2api:main` | `review/fuwei99-main` | `5ee2a1f` | muted 账号检测、账号 active/muted 字段、GHCR workflow、容器配置 bootstrap、新模型映射；2026-05-25 新增 `x-hif-leim` 查询、中文 prompt marker、current-input-file 包装/改名、Docker buildx workflow | `x-hif-leim` 有单点研究价值，但只能按小块验证后引入；中文 prompt marker、`invaild-file*` 包装/改名和测试批量改写风险高，不建议合入。 |
| `ricardosantis/ds2api:main` | `review/ricardosantis-main` | `dc1a76c` | Heroku/Procfile/runtime/env 示例、本地部署文档 | 主要是部署文档和平台样板，价值较低；仅在需要 Heroku 文档时参考。 |
| `tempppw01/ds2api:main` | `review/tempppw01-main` | `9aaa8b4` | 默认禁用上游上传、移除默认 prompt 注入、latest user text prompts、DockerHub workflow | 上游上传禁用已作为可选 runtime 开关吸收；不采用“默认关闭”和 prompt 形状大改，除非有真实回归证据。 |
| `TonyWu2333/ds2api:main` | `review/TonyWu2333-main` | `6dbdcec` | 128k prompt 限制、模型别名微调、prompt role marker 中文化 | 128k 限制已吸收；`6dbdcec` 仅把 DeepSeek marker 换成中文文本，不建议合入，除非动态验证证明更稳。 |
| `1cyberlangke1/dsp:main` | `review/1cyberlangke1-dsp-main` | `0759d7f` | DeepSeek 指纹、device id、x-rangers-id、TLS/transport 调整；同时大规模删除 Vercel/文档/JS 桥 | 冻结为历史快照，不再默认刷新。2026-05-25 与 2026-05-27 两次 GitHub fetch 均返回 `repository not found`；已吸收兼容子集，其余范围偏离 DS2API，不可整合。 |
| `emptysuns/ds2api:main` | `review/emptysuns-main` | `c010373` | prompt/stream/tool 策略；新增 Claude/OpenAI 行为对齐、自动清理 banned 账号、代理批量应用、CI 发布调整 | 旧 stream/tool 候选仍需样本驱动拆分。当前 Claude 已统一走 OpenAI 路径，只审计 body cap 等残余差异；“代理应用到所有账号”已按本仓库 Admin/WebUI 边界实现；自动清理 banned 账号需先用真实错误样本收紧分类。CI 调整无业务价值。 |
| `hefengfan0615/ds2api:main` | `review/hefengfan0615-main` | `bfa9a2d` | 对齐 `deepseek.py` Web 请求外形、标准 HTTP 优先登录、登录错误 body 预览；同时新增默认开启的公共代理池、区域性构建镜像和大体量 Python 参考实现 | 已吸收 completion `action/preempt` 与登录/JSON 诊断预览；Web 请求外形与标准 HTTP 优先登录仍需真实登录/session/PoW/completion 探针 A/B 后再开关化。不要采用默认公共代理、区域性镜像默认值或把 `deepseek.py` 作为运行时资产；历史 Qwen2API 方向仍不整合。 |
| `whatnano/ds2api:main` | `review/whatnano-main` | `b764f1f` | API key 维度 DeepSeek session 复用、prompt prelude 清理/删除、纯文本 prompt、Admin fallback、TLS fingerprint/request jitter、Vercel 代理响应头清理；请求签名已回滚 | Admin fallback 当前主线已有等价实现。session 复用有价值但必须按 stateless API、文件/history split、lease/auto-delete 和 Go/Node 对齐做 opt-in 设计；Vercel 代理响应头清理已按本轮边界排除，TLS/jitter 只适合诊断开关。不要采用删除 role marker/current-input-file/prompt guard 的 prompt 重写。 |
| `voktoylo/ds2api:main` | `review/voktoylo-main` | `ec21187` | `/api/v0/users/current` 主动禁言扫描、`test_status=failed` 账号跳过、`503 upstream_unavailable` 自动切号并标记、账号管理未刷新/刷新所选/错误原因展示 | 已吸收高价值子集：本地长进程后台 `users/current` 禁言扫描复用持久化 `muted/mute_until` 与账号健康冷却，Vercel Node streaming 对上游 `403/429/5xx` 非 200 响应释放 lease 时附带惩罚；仍不整块合入其重复状态源和 UI 大改。 |
| `xiaotian2333/ds2api:main` | `review/xiaotian2333-main` | `6531c90` | 敏感词拦截、`users/current` 禁言检查、插入消息前换行；同时删除 toolcall/toolstream/Node stream sieve 和大量工具调用测试 | 不合入。删除工具调用链路与 OpenAI/Claude/Gemini 协议兼容目标冲突，且 `go test ./...` 当前无法编译；敏感词拦截若需要，应作为独立网关策略重新设计。 |
| `tangsong404/whale2api:main` | `review/tangsong404-whale2api-main` | `cb14c8a` | 独立 DS2API 派生快照；SQLite gateway key/account pool、独立 Pool UI、CSV 账号导入导出、账号禁言/封禁 discard/restore、持久化账号测试任务、自动 discard、flash-only 256K context gate/model advertisement、prompt/tool 中文改写、删减 admin WebUI/Claude/Gemini/pro/search/vision；2026-06-02 增量仅为 CI/test 维护 | 不整块合入。号池运营模型有产品参考价值，但应基于本仓库 `config.Store`/Admin 鉴权或可选持久层重新设计；分类器样本可后续 test-driven 对比。不要吸收 prompt 中文改写、flash-only/pro 删除、协议收缩或硬编码 256K 广告。最新 CI 增量无运行时价值。 |
| `namlevia/ds2api:vi-localization` / `:dev` / `:main` | `review/namlevia-vi-localization` / `review/namlevia-dev` / `review/namlevia-main` | `vi-localization=687dc9f`; `dev=882d0d1`; `main=8316cf8` | 越南语本地化、leaked tool-result 清理；新增密码可见性切换、示例配置清理、默认 Admin key 提示和 dev 抓包 | 密码可见性切换是可独立吸收的小型 UX 候选；示例配置清理只能在保留结构说明前提下参考。不要采用硬编码默认 Admin key、默认 dev 抓包或默认越南语产品入口。`dev` 的 tool-result 清理已吸收。 |
| `Fly143/deepseek-free-api:main` | `review/fly143-main` | `fe88a05` | 独立 Python/FastAPI 实现；Vision `fork_file_task`、动态模型发现；新增模型发现必须带 `did`、空模型列表处理 | Vision `fork_file_task` 仍是优先 live 候选。本仓库当前 `client/settings` 已发送 `did` 和 `scope=model` 且有测试覆盖，本轮模型发现更新无需再移植；其余能力继续只作外部参考。 |

## 当前高价值候选

1. 本地 `C:\Users\highk\Downloads\deepseek-2api-4242` 的账号探针：`/api/v0/users/current` 比 `/api/v0/client/settings` 更适合验证 DeepSeek token；`/api/v0/client/settings?scope=model&did=...` 可读取 `model_configs.value` 与 `vision.switchable`。本仓库已只把这两块吸收到 Admin/WebUI 的账号检查和运行时能力展示，不改变 public 模型目录、模型别名、prompt 兼容流或协议契约。
2. `Fly143/deepseek-free-api` 的 Vision 文件域转换：它在图片上传后调用 `/api/v0/file/fork_file_task`，把普通文件 id 转成 `vision` 可用的文件 id。本仓库已经有 inline image/file 上传、`ref_file_ids`、`model_type=vision` 和文件 ready 等待，但当前没有 `fork_file_task` 路径；这是本次发现的最高价值候选。下一步应先用真实 DeepSeek vision 请求验证：如果现有图片上传在 vision 模型下失败或被忽略，再按 Go client 小块实现 `ForkFileToModelType`，只作用于 vision 图片文件。
3. `whatnano` 的 API-key scoped DeepSeek session reuse：可减少建会话次数并维持同一调用方的上游上下文，但会改变当前 stateless API 语义。只有在明确需求和基准数据下，按 opt-in 方式重新设计，并同时处理首条 prompt 去重、文件引用、history split、账号 lease、auto-delete、并发与 Vercel Node streaming。
4. `hefengfan0615` 的 Web 请求外形与登录诊断：completion 的 `action:null` / `preempt:false`、非 JSON body preview、登录失败/缺 token 脱敏 body 预览已作为低风险诊断小块吸收；随机 Web `device_id`、`os=web`、空 session payload 和标准 HTTP 优先登录仍必须先用真实登录、建会话、PoW、completion 四段探针做 A/B，按可独立回滚的小块引入。
5. `whatnano` 的 Vercel Go 代理响应头清理：去除 `Via`、`X-Forwarded-*` 和平台内部响应头是低风险隐私加固候选，但本轮按用户边界不合入 Vercel 专属链路；TLS fingerprint 与 request jitter 只适合作为实验性诊断开关，不能改变默认请求路径。
6. `fuwei99` 的 `x-hif-leim` 查询：2026-05-25 多次无凭证探针确认 `https://hif-leim.deepseek.com/query` 稳定返回 `code=0` 和 `biz_data.value`。已按 best-effort 方式重新实现到 Go completion/continue 和 Vercel Node streaming：成功则带 `x-hif-leim`，失败则软降级，且有短超时和失败退避。仍缺少真实账号 A/B 数据，后续应验证加 header 前后的 403/429/空回复分布。
7. `emptysuns` 的 stream/SSE 边界修复：包括 UTF-8 replacement 边界、Claude SSE charset、late thinking 抑制、response replacement 在 SSE 边界归一。这些直接影响流式输出质量，但必须用样本回放和三协议测试验证。
8. `namlevia-dev` 的 leaked tool-result section 清理：`3e935c0` 增加 `<|Tool|>...<|end_of_toolresults|>`、全角 `｜` 和 `Assistant_END_OF_TOOL_CALLS` 清理，并覆盖 text/thinking stream chunk 跨段场景。已按本仓库当前结构小块吸收到 shared sanitizer、OpenAI Chat/Responses Go stream runtime 和 Vercel Node bridge，并用 Go/Node 测试覆盖跨 chunk 场景。
9. `emptysuns` 的 tool interception policy：有助于收敛 OpenAI/Claude/Gemini 工具拦截策略，不过会触碰现有 canonical XML tool-call 语义，不能整块搬。
10. `fuwei99` 的容器配置 bootstrap 和模型映射：可能改善容器首次启动体验和模型覆盖，但要先和本仓库 `config`/`model_aliases` 语义对齐。
11. `voktoylo` 的主动禁言扫描和失败账号降权：已按本仓库现有结构吸收 scanner 与 Vercel 非 200 惩罚两小块。未采用上游 `mutestate` 重复状态源、`test_status=failed` 持久跳过和账号管理页大改；这些后续只有在真实样本证明必要时再单独设计。
12. `tangsong404/whale2api` 的号池运营面：SQLite `gateway_api_keys` / `pool_accounts` / `pool_bindings`、CSV 导入导出、key rotation、手动/自动 discard reason、持久化账号测试 job、独立 Pool UI 有产品参考价值。若本仓库需要类似能力，应重新设计到现有 `config.Store`、Admin auth、账号健康冷却和可选持久化后端之上，不直接搬它的独立服务边界。
13. `emptysuns` 的“代理应用到所有账号”和 `namlevia` 的密码可见性切换：都是边界清晰、可单独实现和验证的小型 Admin/WebUI 体验候选，已在本轮落地为后端 Admin 路由、WebUI 操作按钮和密码显隐控件。

### 检测和探针优化

这类内容分散在多个 fork，不能简单归为单一功能合入：

1. 已覆盖或已吸收：`users/current` token/禁言探针、`client/settings?scope=model&did=...` 能力探针、后台禁言扫描、Vercel 非 200 上游惩罚和 `x-hif-leim` best-effort 查询都已经按本仓库 owner 边界落地或有测试覆盖。
2. 仍值得验证：`hefengfan0615` 的标准 HTTP 优先登录、Web request shape，以及 `whatnano` 的 TLS fingerprint/request jitter 诊断开关。`hefengfan0615` 的缺 token 脱敏 body 预览已吸收；其余请求外形项都应先跑真实登录、建会话、PoW、completion 四段 A/B，再决定是否作为可独立回滚的小块引入。
3. 需要样本校准后再做：`emptysuns` 的 banned 自动清理、`tangsong404/whale2api` 的账号测试 job 和 muted/banned 分类器。当前风险是误把瞬时上游错误、content filter 或代理错误归入账号封禁，不能直接触发删除或 discard。
4. 不建议直接采用：默认公共代理、默认持久跳过 `test_status=failed`、重复状态源和整套独立 Pool UI。检测优化应复用本仓库现有账号健康、冷却、Admin/WebUI 和可选持久层，而不是引入第二套状态模型。

### 高价值改造清单

下表按收益、风险、实现边界和验证成本排序。优先级不是 fork 的重要性，而是“值得在本仓库独立开题”的顺序。

| 优先级 | 改造项 | 来源证据 | 本仓库 owner | 验证门槛 |
| --- | --- | --- | --- | --- |
| P0 | Vision 图片上传后执行 `fork_file_task`，把普通 file id 转换成 `vision` 可用 id。 | `Fly143` 在 `proxy.py` 调用 `/api/v0/file/fork_file_task` 并传 `to_model_type=vision`；本仓库已有 upload、ready wait、`ref_file_ids` 和 `model_type=vision`，但没有转换步骤。 | `internal/deepseek/client/client_upload.go`、`internal/deepseek/client/client_file_status.go`、`internal/httpapi/openai/files/*`、`internal/promptcompat/*`。 | 已实现：OpenAI inline upload 解析目标 model type，vision 图片上传 ready 后调用 `/api/v0/file/fork_file_task`，fake transport 单测覆盖请求形状、返回 id 和 gating；仍需真实 DeepSeek vision 样本 A/B。 |
| P0 | 登录/session/PoW/completion 四段 A/B 探针，验证 Web request shape 是否优于当前 Android shape。 | `hefengfan0615` 改为随机 Web `device_id`、`os=web`、空 session body、completion `action:null` / `preempt:false`，并加入登录 body 预览；当前主线已有受保护 `current-android` vs `candidate-web` 四段 live harness、脱敏登录 body 预览和 completion `action/preempt`，默认请求外形仍是 deterministic `DeviceID(accountID)` + `os=Android`、`{"agent":"chat"}`。 | `internal/deepseek/client/client_auth.go`、`internal/deepseek/client/client_http_json.go`、`internal/promptcompat/standard_request.go`，必要时加 `cmd/` 或 `tests/scripts/` 诊断脚本。 | 已用 `admin@fnos` 线上凭据验证：当前池 `86/86` 账号 muted 或未来 `mute_until`，`candidate-web` 前段和稀疏抽样均 0 成功，HTTP 200 小响应体实际是 `biz_code=5 user is muted` JSON 错误。结论是不改默认请求外形，只保留受保护探针和响应体分类。 |
| P1 | 账号失败分类器和安全清理流：先 dry-run 标记 muted/banned/network/rate-limit/content-filter，再人工确认清理。 | `emptysuns` 有 banned 标记和 `/accounts/clean-banned`；`whale2api` 有持久化账号测试 job 与 discard/restore；当前主线已有 muted、health cooldown 和后台扫描。 | `internal/deepseek/client/mute.go`、`internal/deepseek/client/client_auth_helpers.go`、`internal/account/health.go`、`internal/account/mutescan/*`、`internal/httpapi/admin/accounts/*`、`webui/src/features/account/*`。 | 用真实 banned/muted/403/429/5xx/代理失败样本校准分类；默认只标记和展示，不自动删除，不把 content filter 或瞬时上游错误当封禁。 |
| P1 | Vercel Go 代理响应头清理，去除 `Via`、`X-Forwarded-*`、`X-Real-IP`、`Forwarded`、`X-Vercel-*` 等内部拓扑头。 | `whatnano` 在 `internal/js/chat-stream/proxy_go.js` 增加 `shouldStripResponseHeader`；当前主线只过滤 `content-length` / `content-encoding`。 | `internal/js/chat-stream/proxy_go.js` 和对应 Node 测试。 | 本轮按用户边界排除 Vercel 专属改动，暂不合入。 |
| P1 | Admin 代理批量应用到所有账号，带确认和结果计数。 | `emptysuns` 增加 `PUT /admin/proxies/{proxyID}/apply-all` 和 WebUI 操作；当前主线只有单账号代理更新。 | `internal/httpapi/admin/proxies/*`、`internal/config/store_accounts.go`、`webui/src/features/proxy/*`、账号表展示。 | 已实现：Go handler 测试覆盖代理不存在、全部账号更新和 `Pool.Reset()`；WebUI 增加确认操作和结果计数，需通过 build。 |
| P1 | `x-hif-leim` 上游实验从“best-effort 查询”补成可观测 A/B。 | `fuwei99` 给出查询线索；本仓库已实现短超时、失败退避和 Go/Node stream 带 header，但缺真实分布数据。 | `internal/deepseek/client/*hif*`、`internal/js/chat-stream/*`、dev/raw capture 或日志字段。 | 用同账号/同代理对比加 header 前后的 403/429/空回复/首 token 延迟，证明有收益再考虑默认保留；失败仍必须软降级。 |
| P2 | API key scoped DeepSeek session reuse，作为显式 opt-in 实验。 | `whatnano` 复用调用方 API key 维度 session；收益是减少建会话和保留上游上下文，风险是改变 stateless API 语义。 | `internal/auth/*`、`internal/account/*`、`internal/httpapi/openai/chat/*`、`internal/httpapi/openai/history/*`、`internal/js/chat-stream/*`、auto-delete/release 逻辑。 | 先做设计评审和基准。测试必须覆盖默认 stateless 不变、并发调用方隔离、文件引用、history split、auto-delete、lease 释放、Go/Node streaming 对齐。 |
| P2 | TLS fingerprint/request jitter 诊断开关，仅用于上游风控排查。 | `whatnano` 增加 TLS fingerprint 和 request jitter；这类改动可能改变全局上游行为。 | `internal/deepseek/transport/*`、`internal/deepseek/client/*`、runtime/settings 配置。 | 默认关闭；只在探针脚本和 debug 配置中启用，验证成功率/延迟/错误码后再决定是否产品化。 |
| 小成本 | 密码可见性切换和代理/账号表单 UX 小修。 | `namlevia` 的密码可见性切换独立、边界小；不绑定其默认 Admin key、dev capture 或语言切换。 | `webui/src/features/account/*`、`webui/src/features/proxy/*`、登录页 Admin Key 字段。 | 已实现：登录、添加账号、代理表单新增显隐切换；需通过 `npm run build --prefix webui`。 |

## 当前不建议合入

1. `TonyWu2333` 的中文 prompt marker：修改核心 prompt 形状，缺少动态证据时风险高。
2. `tempppw01` 的默认禁用上游上传：本仓库已保留为可选开关，默认关闭会改变兼容行为。
3. `1cyberlangke1/dsp` 的大范围删减：删除 Vercel、文档和 Node stream 桥，不符合本仓库架构边界。
4. `hefengfan0615` 的工具调用禁用链路：会让 `tools` 注入、OpenAI `tool_calls`、Responses function-call 事件和 `DS2API_TOOLS.txt` 上传退化为纯文本，和当前协议兼容目标冲突。
5. `hefengfan0615` 删除 Output integrity guard：属于 prompt 形状削弱，只有在动态样本证明 guard 本身导致稳定回归时才考虑开关化，而不是直接删除。
6. `hefengfan0615` 的 Qwen2API 改造：包括登录 payload、session payload、completion payload、忽略 PoW、Qwen SSE 分支和模型目录的成套替换。它是另一个上游服务适配，不是 DS2API 的 DeepSeek upstream 修复。
7. `fuwei99` 的中文 prompt marker 和 `invaild-file*` current-input-file 包装/改名：会改变核心 prompt 形状和 file-reference 语义，且分支通过批量测试改写适配自身实现，不能证明对本仓库有效。
8. 未记录 fork 中的个人配置/凭证提交：`lamthien8x` 和 `dangtai2710` 当前仍主要是 `config.json` / `config.example.json` 中的个人配置或明文凭据改动，不能吸收。
9. `xiaotian2333` 的工具调用删除链路：大规模删除 `internal/toolcall`、`internal/toolstream`、Node stream sieve、prompt tool injection 和协议测试，会直接破坏当前工具调用兼容契约；不能用“禁用工具”方式修复上游风险。
10. `tangsong404/whale2api` 的 prompt/tool 中文改写、flash-only 模型收缩、`deepseek-v4-pro`/文件上传/long-history 路径删除、以及硬编码 256K context advertisement：这些都和本仓库的 OpenAI/Claude/Gemini + Responses/files 多协议兼容目标冲突。其 probe 把部分瞬时错误视为 OK、把 content filter 归入 banned，也不能在没有真实样本校准前直接采用。
11. `namlevia/vi-localization` 的整块越南语本地化和默认 README 切换：这是产品/文档语言方向，不是上游 DeepSeek 行为修复；其中 `start.mjs` 大幅重写、`README.MD` 默认语言切换和独立 Docker workflow 不应随 tool-result 清理一起合入。
12. `Fly143/deepseek-free-api` 的整块 Python/FastAPI runtime、BasicAuth Admin、CORS/i18n/Docker workflow、多账号/代理/token refresh 重写：本仓库已有 Go runtime、React Admin、配置热更新、账号池、代理、PoW、CORS 和部署边界，不能整块替换。
13. `Fly143/deepseek-free-api` 的工具 passthrough/no-tools 分支和默认 context pruning：前者绕开本仓库 canonical XML tool-call 与 Go/Node stream anti-leak 契约，后者会删除、截断或清空 prompt-visible 历史内容；只能在真实失败样本证明必要后做开关化小块验证。
14. `whatnano` 的纯文本 prompt、删除 prompt prelude/role marker/current-input-file 和 Output integrity guard：会破坏当前 prompt compatibility、工具调用、文件引用与历史拆分契约。该分支的 `X-Client-*` 请求签名也已自行回滚，没有吸收价值。
15. `hefengfan0615` 的默认开启公共代理池、区域性 GOPROXY/npm 镜像默认值和仓库内大体量 `deepseek.py`：公共代理存在隐私、可用性和安全风险，镜像默认值不适合作为跨区域项目默认，参考脚本不应成为运行时资产。
16. `namlevia/vi-localization` 的硬编码默认 Admin key、默认开启 dev packet capture 和默认越南语入口：会降低安全默认值或改变产品语言边界，不能随小型 UI 改进一起引入。

## 2026-06-12 增量复查

本次按文档常规 fetch 清单刷新全部远端 ref，并新增 `whatnano/ds2api:main`。`tangsong404/whale2api` 首次 fetch 遇到 TLS EOF，重试后成功。

- 未变化：`CJackHwang` `8316cf8`、`qingdeng888` `42bc1ed`、`fuwei99` `5ee2a1f`、`ricardosantis` `dc1a76c`、`tempppw01` `9aaa8b4`、`TonyWu2333` `6dbdcec`、`voktoylo` `ec21187`、`xiaotian2333` `6531c90`、`tangsong404/whale2api` `cb14c8a`、`namlevia/main` `8316cf8`、`namlevia/dev` `882d0d1`。
- 有更新：`emptysuns` 从 `2f87937` 到 `c010373`，`hefengfan0615` 从 `8316cf8` 到 `bfa9a2d`，`namlevia/vi-localization` 从 `91bb723` 到 `687dc9f`，`Fly143` 从 `93cf744` 到 `fe88a05`；新增 `whatnano` `b764f1f`。
- `whatnano`：相对官方基线包含 API key 维度 session reuse、prompt 大幅简化、Admin fallback、TLS fingerprint、request jitter 和 Vercel Go 代理响应头清理。当前主线已独立实现 Admin fallback；session reuse 只能作为 opt-in 专题重设计；Vercel 响应头清理本轮按用户边界暂不合入；纯文本 prompt 和删除 current-input-file/guard 与本仓库协议兼容契约冲突，不合入。其请求签名提交已在 fork 内回滚。
- `hefengfan0615`：当前增量重新回到 DeepSeek Web 请求外形方向，包含登录/session/completion payload、header、标准 HTTP 优先登录和错误 body 预览，并混入默认公共代理池、区域性构建镜像和 `deepseek.py`。本轮只吸收脱敏错误预览、非 JSON 响应预览、completion `action/preempt` 和受保护 `current-android` vs `candidate-web` 四段 live harness；`admin@fnos` 线上凭据验证未证明 `candidate-web` 可降低封禁或绕过 mute，因此默认请求外形与标准 HTTP 优先登录不采用。默认公共代理和镜像也不采用。
- `emptysuns`：新增 Claude/OpenAI 对齐、自动清理 banned 账号和代理批量应用。当前主线 Claude 已统一代理到 OpenAI 路径，不能直接搬旧 handler 对齐补丁；代理批量应用可独立实现。自动清理需先用真实 banned 登录响应校准分类，避免误删账号。
- `namlevia/vi-localization`：新增密码可见性切换、示例配置清理、默认 Admin key 提示和 dev packet capture。仅密码可见性切换适合直接作为小型 UX 候选；安全默认值和产品语言改动不采用。
- `Fly143`：新增模型发现必须传 `did` 和空模型列表处理。本仓库当前 `internal/deepseek/client/client_probe.go` 已发送 `did` 与 `scope=model`，并由测试覆盖，无需移植；Vision `fork_file_task` 仍是其唯一高优先级 live 候选。

进一步本地验证：

- `whatnano` 的 Admin deep-link fallback 已由当前主线覆盖，并新增 `internal/server/router_webui_test.go` 的 `TestAdminDeepLinkFallsBackToWebUIIndex` 固化：未知 `GET /admin/...` 路径会回退到 WebUI `index.html`，且使用 `no-store, must-revalidate`。
- `Fly143` 的模型发现 `did`/`scope=model` 要求已由 `internal/deepseek/client/client_probe_test.go` 的 `TestGetAccountCapabilitiesParsesModelConfigsArray` 覆盖；本轮无需改动 `client_probe.go`。
- 静态读回确认当前主线仍未吸收 `hefengfan0615` 的默认请求外形变更：登录仍是 deterministic `DeviceID(accountID)` + `os=Android`，建会话 body 仍是 `{"agent":"chat"}`，未启用 Web headers、随机 device id、`os=web`、空 session body 或标准 HTTP 优先登录。已吸收的是 completion `action:null` / `preempt:false`、登录缺 token/失败的脱敏 body 预览、非 JSON 上游响应预览，以及受保护的 `current-android` vs `candidate-web` 四段 live 探针。后续用 `admin@fnos` 线上配置补跑后，`candidate-web` 对前段和稀疏账号抽样均无真实 SSE 内容，HTTP 200 小响应体被确认是 `biz_code=5 user is muted`，因此不进入正式运行时代码。
- 静态读回确认当前 Vercel Go 代理仍只过滤 `content-length` / `content-encoding`，未过滤 `Via`、`X-Forwarded-*` 或平台内部响应头；本轮按用户边界不合入 Vercel 专属链路。
- 已运行 `go test ./internal/deepseek/client ./internal/httpapi/openai ./internal/httpapi/admin/proxies ./internal/server ./internal/promptcompat`、`npm run build --prefix webui`，并用浏览器验证登录页密码显隐控件。未运行真实 DeepSeek 登录/session/PoW/completion 或 vision A/B；这些会触达真实账号和上游状态，需在明确测试账号与目标样本后单独执行。

## 2026-06-03 增量复查

本次新增 `Fly143/deepseek-free-api` 监控 ref：

- `git ls-remote --symref https://github.com/Fly143/deepseek-free-api HEAD` 确认默认分支为 `main`，HEAD `93cf744`；已添加本地 remote `fly143`，并拉取到 `refs/remotes/review/fly143-main`。
- `git merge-base main review/fly143-main` 无共同祖先；该项目是 Python/FastAPI + `curl_cffi` Chrome impersonation 的独立实现，不是本仓库 Go 代码的可 cherry-pick fork。
- 主要功能：OpenAI Chat/Responses、Structured Output、Anthropic Messages/Batches、动态模型发现、Vision 图片上传、Responses 本地 JSON 持久化与 compact、900K 左右 context pruning、`/v1/v1` 双前缀兼容、BasicAuth Admin、CORS/i18n/Docker、Node WASM PoW + Python fallback、多账号 round-robin。
- 和本仓库已对齐的能力：本仓库已有 Go PoW、账号池/队列、代理、Admin/WebUI、CORS、OpenAI/Claude/Gemini/Responses 路由、inline file/image 上传、`ref_file_ids`、DeepSeek `client/settings?scope=model` 能力探针、long-history split、tool XML/stream anti-leak 和 Vercel Node bridge。
- 高价值缺口只锁定一项：Fly143 对图片上传后额外调用 `/api/v0/file/fork_file_task` 并指定 `to_model_type=vision`。本仓库当前只有上传和 ready 等待，没有该转换；应先用真实 vision 请求复现当前行为，再决定是否实现 Go client 小块。
- 条件参考项：`/v1/v1` 兼容中间件成本低，可在有真实客户端 404 样本时加入；Responses 持久化/compact 和 Anthropic batches 有客户端兼容价值，但必须按本仓库 owner-scoped store、TTL、Admin/用户鉴权、取消/队列语义重新设计，不能直接照搬 Fly143 的本地 JSON/background task。
- 不建议合入：整块 Python runtime、BasicAuth Admin、CORS/i18n/Docker、工具 passthrough/no-tools 分支和默认 context pruning。这些要么本仓库已有更贴合架构的实现，要么会改变核心 prompt/tool/history 契约。

## 2026-06-02 增量复查

本次刷新已监控 ref 并新增 `namlevia/ds2api`：

- `hefengfan0615/ds2api:main`：当前 HEAD `8316cf8`，tree hash `72eca1d867a5247ea846ab1065085843778124ad`，与 `CJackHwang/main` 完全一致。旧的 Qwen2API 发散内容不再存在于当前 `main`，没有当前可吸收增量。
- `tangsong404/whale2api:main`：从 `7d703aa` 更新到 `cb14c8a`，新增提交为 `79ac467`、`ad96331`、`6f71c09`、`124a576`、`cb14c8a`。差异仅涉及 `.github/workflows/docker.yml`、`tests/scripts/run-unit-all.sh`、`tests/scripts/run-unit-node.sh` 和删除其仓库里的 `tests/node/chat-history-utils.test.js`；属于 CI/test 维护，无运行时或协议价值。
- `namlevia/ds2api`：`git ls-remote --symref` 确认默认分支为 `vi-localization`，另有 `dev` 和 `main`。`main` 为 `8316cf8`，tree hash 与官方基线相同。
- `namlevia/vi-localization`：HEAD `91bb723`，相对官方基线新增 3 个提交：`a95877d` 越南语 WebUI/README 本地化、`f526864` 把默认 README 切成越南语、`91bb723` 增加 multi-arch GHCR Docker workflow。`git diff --stat refs/remotes/review/cjack-main..refs/remotes/review/namlevia-vi-localization` 显示 12 个文件、2102 行新增、883 行删除，主要是 README、`webui/src/locales/vi.json`、`webui/src/i18n.jsx`、`start.mjs` 和 Docker workflow。
- `namlevia/dev`：HEAD `882d0d1`，相对官方基线有 `3e935c0 fix(openai): strip leaked tool result markers`、合并提交 `92b3093` 和 `VERSION` 更新 `882d0d1`。核心补丁改动 4 个 Go test/实现文件，增加完整 tool-result section 和 stream chunk 跨段清理测试；`VERSION` 更新本身没有可吸收价值。
- 结论：把 `namlevia` 加入长期监控。默认分支的越南语本地化和 CI workflow 不建议合入；`dev` 中 leaked tool-result section 清理是唯一有运行时价值的小块，本次已按本仓库当前 `chat_stream_runtime.go`、`responses_stream_runtime_core.go` 和 Vercel Node bridge 重新实现 Go/Node 对齐，而不是直接搬上游 `stream_accumulator.go` 结构。

## 2026-05-29 增量复查

本次新增 `tangsong404/whale2api` 监控 ref：

- `tangsong404/whale2api`：`7d703aa`。`git merge-base refs/remotes/review/cjack-main refs/remotes/review/tangsong404-whale2api-main` 无共同祖先，按独立 DS2API 派生快照处理，而不是普通 fork 增量。
- README 声明该项目基于 DS2API 改名为 Whale2API，只支持 OpenAI Chat Completions 兼容，主打 `deepseek-v4-flash` 256K 上下文和大号池使用；作者明确写到移除 DS2API prompt 文字、中文 prompt/tool 符号“经常导致出错”、不再使用 `deepseek-v4-pro`，并新增禁言而非封禁的检测思路。
- 代码证据：`internal/pooldb/migrations/001_gateway_pool.sql` 和 `003_account_test_jobs.sql` 定义 SQLite gateway key/account pool/binding 与持久化账号测试任务；`internal/poolui/server.go` 提供独立 Pool UI API；`internal/pooldb/admin_ops.go` 实现 CSV 导入导出、key 管理和 discard/restore；`internal/accountprobe/probe.go` 与 `internal/poolaccounthealth/classify.go` 做登录/补全探针和 muted/banned 分类；`internal/completionruntime/context_limit.go` 与 `internal/config/models.go` 写死 flash-only 256K 用户可见 context；`internal/server/router.go` 保留 OpenAI chat/responses/files/embeddings 路由，但没有本仓库的 admin WebUI、Claude、Gemini、pro/search/vision 路径。
- 验证：`git fetch https://github.com/tangsong404/whale2api +refs/heads/main:refs/remotes/review/tangsong404-whale2api-main` 成功；隔离 worktree 执行 `go test ./...` 全部通过。
- 结论：有价值的新特性主要是“号池运营产品面”，包括持久化 gateway key -> account binding、CSV 批量运维、独立 Pool UI、discard reason 和持久化测试 job。该方向若要进入本仓库，需要先按现有 Admin/账号健康/配置热更新模型重设计；不建议直接合入其独立 SQLite 服务边界，也不建议采用 prompt 中文改写、flash-only/pro 删除、协议收缩或硬编码 256K 上下文广告。

## 2026-05-27 增量复查

已刷新长期监控 ref：

- 未变化：`CJackHwang` `8316cf8`、`qingdeng888` `42bc1ed`、`fuwei99` `5ee2a1f`、`ricardosantis` `dc1a76c`、`tempppw01` `9aaa8b4`、`TonyWu2333` `6dbdcec`、`emptysuns` `2f87937`、`hefengfan0615` `37f6206`。
- 无法刷新：`1cyberlangke1/dsp` 远端仍返回 `repository not found`，已从常规 fetch 清单移除；本地 `0759d7f` 只保留为旧快照。

本次新增长期监控 fork：

- `voktoylo/ds2api`：`ec21187`。相对 `CJackHwang/main` 有 7 个提交、54 个文件，主要新增 `internal/account/mutescan`、`internal/account/mutestate`、`/api/v0/users/current` 账号禁言探针、pool 跳过 failed/muted 账号、`503 upstream_unavailable` 切号标记，以及账号管理页的状态分组和刷新所选。
  - 验证：隔离 worktree 跑 `go test ./...` 通过；`npm run build --prefix webui` 通过；`git diff --check refs/remotes/review/cjack-main...refs/remotes/review/voktoylo-main` 未通过，问题集中在 `webui/src/features/account/useAccountsData.js` 的尾随空白。
  - 结论：有价值但只适合拆分吸收。本仓库已经有持久化 `muted/mute_until`、reactive mute detection、账号健康冷却和账号 probe，直接合入会重复状态源；已吸收主动 `users/current` 扫描和 Vercel 上游非 200 惩罚，silent block/UI 错误原因仍留作后续专题。
- `xiaotian2333/ds2api`：`6531c90`。相对 `CJackHwang/main` 有 6 个提交、126 个文件，新增 `internal/sensitivewords` 和 `users/current` 禁言检查，但同时删除工具调用、toolstream、Node stream sieve、工具 prompt 注入和大量协议测试。
  - 验证：`git diff --check refs/remotes/review/cjack-main...refs/remotes/review/xiaotian2333-main` 通过；隔离 worktree 跑 `go test ./...` 未通过，代表性错误包括 `promptcompat.DefaultToolChoicePolicy`、`ToolChoicePolicy`、`buildClaudeToolPrompt`、`buildOpenAIFinalPrompt` 和 `Turn.ToolCalls` 被删除后仍被测试/调用引用。
  - 结论：不纳入可合入候选。敏感词拦截可以作为独立产品策略重新设计，但不能接受以删除工具调用兼容层为代价的分支。

上次未确认的短期 fork 本次已拉取到本地：

- `6um6n7qu/ds2api`：`85d8f7f`。新增的是另一套占位入口和 `config` 包，包含 `github.com/yourusername/ds2api/internal/api` 这类不匹配本仓库 module 的 import，并混入 Synology NAS 配置语义；不纳入监控。隔离 worktree 跑 `go test ./...` 未通过，根包在 setup 阶段失败。
- `maple323/ds2api`：`f1f7dd8`。只把 `.env.example` 改名成 `config.json`，内容仍是 env 格式并含示例 admin key；`git diff --check refs/remotes/review/cjack-main...HEAD` 通过，但没有可吸收业务价值。
- `zzz449/ds2api`：`28b9b5b`。删除 Output integrity guard，并把 `.gitignore` 改成带 Markdown fence 的通用模板；隔离 worktree 跑 `go test ./internal/prompt` 编译失败，`messages_test.go` 仍引用被删除的 `outputIntegrityGuardPrompt`，不纳入监控。

## 2026-05-25 增量复查

已刷新长期监控 ref：

- 未变化：`CJackHwang` `8316cf8`、`qingdeng888` `42bc1ed`、`ricardosantis` `dc1a76c`、`tempppw01` `9aaa8b4`、`TonyWu2333` `6dbdcec`、`emptysuns` `2f87937`。
- 有更新：`fuwei99` 从 `08b4aa5` 到 `5ee2a1f`；`hefengfan0615` 从本地旧 ref `2fd3c0d` 到 `37f6206`。
- 无法刷新：`1cyberlangke1/dsp` 远端返回 repository not found，本地 `0759d7f` 保留为旧快照；2026-05-27 复查后已从常规 fetch 清单移除。

`fuwei99` 增量结论：

- `x-hif-leim` endpoint 本身有效，返回可用于 header 的 token 字符串；12 次连续探针均成功，首个请求约 225ms，后续约 31-41ms。
- 本仓库已重新实现该小块，但没有照搬上游：只在 completion/continue 路径 best-effort 查询，失败不会阻断请求；Go 和 Vercel Node runtime 保持对齐。
- 中文 role marker、清洗 `[system]`/`[user]`/`[assistant]` 文本、`invaild-file*` 文件名和 wrapper 均会改变 prompt/file contract；没有 live 证据前不采纳。
- Docker workflow 改为 buildx/cache 属于 CI 便利项；本仓库已有 release/quality gate 工作流，价值低于业务兼容候选。

`hefengfan0615` 增量结论：

- 新增 Qwen2API 方向已经不是 DeepSeek Web upstream 的小修，而是把登录、session、completion、PoW、模型目录、SSE parser 一起改成 Qwen 假设。
- `internal/sse/parser.go` 的 Qwen 分支替换了 DeepSeek `p/v` 解析路径，按本仓库目标会直接破坏现有 DeepSeek SSE。
- `/v1/models` 按账号返回模型和 WebUI 根据 key/account 重拉模型的想法可以单独立项，但该实现绑定 Qwen、绕开 `authFetch`，不能直接搬。

未记录 fork 复查：

- `lamthien8x/ds2api`：`61723a6`，主要提交个人 `config.json`，且取消忽略本地配置文件；不纳入监控。
- `kingleykin/ds2api`：`e8eea7d`，主要是越南语 `START_HERE` 和示例配置；不纳入监控。
- `dangtai2710/ds2api`：`dd730aa`，只改 `config.example.json` 中示例账号字段；不纳入监控。
- `leeamkim/ds2api`：`379d85a`，是 Synology/NAS 私有客户端骨架和 timeout 调整，和本仓库 DeepSeek/OpenAI/Claude/Gemini 适配主线无关；不纳入监控。
- `6um6n7qu/ds2api`、`maple323/ds2api`、`zzz449/ds2api`：GitHub API rate limit 后，本机 `git ls-remote` 多次返回 TLS EOF，本次未能确认远端 HEAD；保持“不纳入长期监控”的旧结论。2026-05-27 已重新拉取并复查，仍无可吸收价值。

## 2026-05-19 其他上游快扫

已记录监控对象中，除 `1cyberlangke1/dsp` 从 `10f0ca3` 非快进更新到 `0759d7f` 外，`CJackHwang`、`qingdeng888`、`fuwei99`、`ricardosantis`、`tempppw01`、`TonyWu2333`、`emptysuns` 的 HEAD 均未变化。

GitHub forks API 按近期 push 快扫到的未记录 fork：`6um6n7qu/ds2api`、`maple323/ds2api`、`zzz449/ds2api`、`lamthien8x/ds2api`、`dangtai2710/ds2api`、`kingleykin/ds2api`、`leeamkim/ds2api`。这些分支主要是个人配置文件、占位 `main.go`/`config` 包、Synology 私有客户端、README/START_HERE 文档或同样删除 Output integrity guard；未发现比当前监控清单更值得吸收的通用 DS2API 实现，暂不纳入长期监控。
