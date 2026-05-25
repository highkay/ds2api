# 上游 Fork 监控清单

本文记录需要定期跟踪的 DS2API 上游和活跃 fork，用于复查可吸收的实现、规避重复调研，并给出当前处理建议。

最近复查时间：2026-05-25。

## 本地引用

所有监控分支统一拉到 `refs/remotes/review/*`，避免污染正式 remote。

```bash
git fetch https://github.com/CJackHwang/ds2api +refs/heads/main:refs/remotes/review/cjack-main
git fetch https://github.com/qingdeng888/ds2api +refs/heads/main:refs/remotes/review/qingdeng888-main
git fetch https://github.com/fuwei99/ds2api +refs/heads/main:refs/remotes/review/fuwei99-main
git fetch https://github.com/ricardosantis/ds2api +refs/heads/main:refs/remotes/review/ricardosantis-main
git fetch https://github.com/tempppw01/ds2api +refs/heads/main:refs/remotes/review/tempppw01-main
git fetch https://github.com/TonyWu2333/ds2api +refs/heads/main:refs/remotes/review/TonyWu2333-main
git fetch https://github.com/1cyberlangke1/dsp +refs/heads/main:refs/remotes/review/1cyberlangke1-dsp-main
git fetch https://github.com/emptysuns/ds2api +refs/heads/main:refs/remotes/review/emptysuns-main
git fetch https://github.com/hefengfan0615/ds2api +refs/heads/main:refs/remotes/review/hefengfan0615-main
```

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
| `1cyberlangke1/dsp:main` | `review/1cyberlangke1-dsp-main` | `0759d7f` | DeepSeek 指纹、device id、x-rangers-id、TLS/transport 调整；同时大规模删除 Vercel/文档/JS 桥 | 只监控上游指纹相关小块；已吸收兼容子集。该 fork 范围偏离 DS2API，不可整合。2026-05-25 复查时 GitHub fetch 返回 repository not found，当前 ref 只能视为本地旧快照。 |
| `emptysuns/ds2api:main` | `review/emptysuns-main` | `2f87937` | prompt 默认策略、response replacements、tool interception 统一、SSE/stream 边界清洗、late thinking 抑制、WebUI 设置面板 | 有研究价值但改动面很大。建议作为单独专题审查：先用 raw SSE/协议测试证明具体问题，再按 `responserewrite`、`toolpolicy`、`sse normalizer` 等小块拆分引入。 |
| `hefengfan0615/ds2api:main` | `review/hefengfan0615-main` | `37f6206` | 随机 device_id、随机 DeepSeek header/TLS 指纹；后续继续发散为 Qwen2API 改造、Qwen SSE 解析、忽略 PoW、账号相关 `/v1/models` 和 WebUI 模型加载调整 | 不整合。新增 Qwen2API 方向已偏离 DS2API 的 DeepSeek upstream owner，并替换/绕过 DeepSeek SSE、PoW、登录和模型契约；账号相关模型列表 UI 可以作为独立需求重新设计，但不能从该分支直接搬。详见 [hefengfan0615 上游评估报告](./upstream-hefengfan0615-evaluation.md)。 |

## 当前高价值候选

1. `fuwei99` 的 `x-hif-leim` 查询：2026-05-25 多次无凭证探针确认 `https://hif-leim.deepseek.com/query` 稳定返回 `code=0` 和 `biz_data.value`。已按 best-effort 方式重新实现到 Go completion/continue 和 Vercel Node streaming：成功则带 `x-hif-leim`，失败则软降级，且有短超时和失败退避。仍缺少真实账号 A/B 数据，后续应验证加 header 前后的 403/429/空回复分布。
2. `emptysuns` 的 stream/SSE 边界修复：包括 UTF-8 replacement 边界、Claude SSE charset、late thinking 抑制、response replacement 在 SSE 边界归一。这些直接影响流式输出质量，但必须用样本回放和三协议测试验证。
3. `emptysuns` 的 tool interception policy：有助于收敛 OpenAI/Claude/Gemini 工具拦截策略，不过会触碰现有 canonical XML tool-call 语义，不能整块搬。
4. `hefengfan0615` 早期请求外形/指纹方向：可能对 DeepSeek 上游风控有帮助，但必须先用真实登录、建会话、PoW、completion 四段探针验证成功率和错误分布；不要直接引入 `math/rand` 全局随机、不稳定 TLS 指纹，或后续 Qwen2API 改造。
5. `fuwei99` 的容器配置 bootstrap 和模型映射：可能改善容器首次启动体验和模型覆盖，但要先和本仓库 `config`/`model_aliases` 语义对齐。

## 当前不建议合入

1. `TonyWu2333` 的中文 prompt marker：修改核心 prompt 形状，缺少动态证据时风险高。
2. `tempppw01` 的默认禁用上游上传：本仓库已保留为可选开关，默认关闭会改变兼容行为。
3. `1cyberlangke1/dsp` 的大范围删减：删除 Vercel、文档和 Node stream 桥，不符合本仓库架构边界。
4. `hefengfan0615` 的工具调用禁用链路：会让 `tools` 注入、OpenAI `tool_calls`、Responses function-call 事件和 `DS2API_TOOLS.txt` 上传退化为纯文本，和当前协议兼容目标冲突。
5. `hefengfan0615` 删除 Output integrity guard：属于 prompt 形状削弱，只有在动态样本证明 guard 本身导致稳定回归时才考虑开关化，而不是直接删除。
6. `hefengfan0615` 的 Qwen2API 改造：包括登录 payload、session payload、completion payload、忽略 PoW、Qwen SSE 分支和模型目录的成套替换。它是另一个上游服务适配，不是 DS2API 的 DeepSeek upstream 修复。
7. `fuwei99` 的中文 prompt marker 和 `invaild-file*` current-input-file 包装/改名：会改变核心 prompt 形状和 file-reference 语义，且分支通过批量测试改写适配自身实现，不能证明对本仓库有效。
8. 未记录 fork 中的个人配置/凭证提交：`lamthien8x` 和 `dangtai2710` 当前仍主要是 `config.json` / `config.example.json` 中的个人配置或明文凭据改动，不能吸收。

## 2026-05-25 增量复查

已刷新长期监控 ref：

- 未变化：`CJackHwang` `8316cf8`、`qingdeng888` `42bc1ed`、`ricardosantis` `dc1a76c`、`tempppw01` `9aaa8b4`、`TonyWu2333` `6dbdcec`、`emptysuns` `2f87937`。
- 有更新：`fuwei99` 从 `08b4aa5` 到 `5ee2a1f`；`hefengfan0615` 从本地旧 ref `2fd3c0d` 到 `37f6206`。
- 无法刷新：`1cyberlangke1/dsp` 远端返回 repository not found，本地 `0759d7f` 保留为旧快照。

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
- `6um6n7qu/ds2api`、`maple323/ds2api`、`zzz449/ds2api`：GitHub API rate limit 后，本机 `git ls-remote` 多次返回 TLS EOF，本次未能确认远端 HEAD；保持“不纳入长期监控”的旧结论，但下次可从 GitHub forks API 重新拉取。

## 2026-05-19 其他上游快扫

已记录监控对象中，除 `1cyberlangke1/dsp` 从 `10f0ca3` 非快进更新到 `0759d7f` 外，`CJackHwang`、`qingdeng888`、`fuwei99`、`ricardosantis`、`tempppw01`、`TonyWu2333`、`emptysuns` 的 HEAD 均未变化。

GitHub forks API 按近期 push 快扫到的未记录 fork：`6um6n7qu/ds2api`、`maple323/ds2api`、`zzz449/ds2api`、`lamthien8x/ds2api`、`dangtai2710/ds2api`、`kingleykin/ds2api`、`leeamkim/ds2api`。这些分支主要是个人配置文件、占位 `main.go`/`config` 包、Synology 私有客户端、README/START_HERE 文档或同样删除 Output integrity guard；未发现比当前监控清单更值得吸收的通用 DS2API 实现，暂不纳入长期监控。
