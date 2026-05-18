# 上游 Fork 监控清单

本文记录需要定期跟踪的 DS2API 上游和活跃 fork，用于复查可吸收的实现、规避重复调研，并给出当前处理建议。

最近复查时间：2026-05-18。

## 本地引用

所有监控分支统一拉到 `refs/remotes/review/*`，避免污染正式 remote。

```bash
git fetch https://github.com/CJackHwang/ds2api refs/heads/main:refs/remotes/review/cjack-main
git fetch https://github.com/qingdeng888/ds2api refs/heads/main:refs/remotes/review/qingdeng888-main
git fetch https://github.com/fuwei99/ds2api refs/heads/main:refs/remotes/review/fuwei99-main
git fetch https://github.com/ricardosantis/ds2api refs/heads/main:refs/remotes/review/ricardosantis-main
git fetch https://github.com/tempppw01/ds2api refs/heads/main:refs/remotes/review/tempppw01-main
git fetch https://github.com/TonyWu2333/ds2api refs/heads/main:refs/remotes/review/TonyWu2333-main
git fetch https://github.com/1cyberlangke1/dsp refs/heads/main:refs/remotes/review/1cyberlangke1-dsp-main
git fetch https://github.com/emptysuns/ds2api refs/heads/main:refs/remotes/review/emptysuns-main
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
| `fuwei99/ds2api:main` | `review/fuwei99-main` | `08b4aa5` | muted 账号检测、账号 active/muted 字段、GHCR workflow、容器配置 bootstrap、新模型映射 | muted 检测与 GHCR 核心已吸收；模型映射和 bootstrap 需按本仓库配置语义逐项复核，不整分支合并。 |
| `ricardosantis/ds2api:main` | `review/ricardosantis-main` | `dc1a76c` | Heroku/Procfile/runtime/env 示例、本地部署文档 | 主要是部署文档和平台样板，价值较低；仅在需要 Heroku 文档时参考。 |
| `tempppw01/ds2api:main` | `review/tempppw01-main` | `9aaa8b4` | 默认禁用上游上传、移除默认 prompt 注入、latest user text prompts、DockerHub workflow | 上游上传禁用已作为可选 runtime 开关吸收；不采用“默认关闭”和 prompt 形状大改，除非有真实回归证据。 |
| `TonyWu2333/ds2api:main` | `review/TonyWu2333-main` | `6dbdcec` | 128k prompt 限制、模型别名微调、prompt role marker 中文化 | 128k 限制已吸收；`6dbdcec` 仅把 DeepSeek marker 换成中文文本，不建议合入，除非动态验证证明更稳。 |
| `1cyberlangke1/dsp:main` | `review/1cyberlangke1-dsp-main` | `10f0ca3` | DeepSeek 指纹、device id、x-rangers-id、TLS/transport 调整；同时大规模删除 Vercel/文档/JS 桥 | 只监控上游指纹相关小块；已吸收兼容子集。该 fork 范围偏离 DS2API，不可整合。 |
| `emptysuns/ds2api:main` | `review/emptysuns-main` | `2f87937` | prompt 默认策略、response replacements、tool interception 统一、SSE/stream 边界清洗、late thinking 抑制、WebUI 设置面板 | 有研究价值但改动面很大。建议作为单独专题审查：先用 raw SSE/协议测试证明具体问题，再按 `responserewrite`、`toolpolicy`、`sse normalizer` 等小块拆分引入。 |

## 当前高价值候选

1. `emptysuns` 的 stream/SSE 边界修复：包括 UTF-8 replacement 边界、Claude SSE charset、late thinking 抑制、response replacement 在 SSE 边界归一。这些直接影响流式输出质量，但必须用样本回放和三协议测试验证。
2. `emptysuns` 的 tool interception policy：有助于收敛 OpenAI/Claude/Gemini 工具拦截策略，不过会触碰现有 canonical XML tool-call 语义，不能整块搬。
3. `fuwei99` 的容器配置 bootstrap 和模型映射：可能改善容器首次启动体验和模型覆盖，但要先和本仓库 `config`/`model_aliases` 语义对齐。

## 当前不建议合入

1. `TonyWu2333` 的中文 prompt marker：修改核心 prompt 形状，缺少动态证据时风险高。
2. `tempppw01` 的默认禁用上游上传：本仓库已保留为可选开关，默认关闭会改变兼容行为。
3. `1cyberlangke1/dsp` 的大范围删减：删除 Vercel、文档和 Node stream 桥，不符合本仓库架构边界。

