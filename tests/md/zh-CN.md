---
title: "发布检查清单"
description: "在同步本地化内容之前，请验证文档更新。"
---

# 发布检查清单

在不破坏占位符（例如 `HLMDPH_*`）或标志（例如 `HLMDPH_13FC0B106E5F_1`）的情况下，更新发布。

Use the [status reference](https://example.com/docs/status?tab=cli#dry-run) before you push changes.

Reference links should also survive: [CLI guide][cli-guide] and ![Diagram](https://example.com/assets/flow(chart).png).

> 保持重复标签不变。
> 保持重复标签不变。
>
> Preserve `MDPH_0_END` as literal prose, not as a parser token.

- 在终端中查看“同步摘要”。
- Confirm links in [Troubleshooting](https://example.com/docs/troubleshooting#common-errors) stay intact.
- 不要翻译`hyperlocalise run --group docs`。
- 小心处理转义字符，例如 ``\*literal asterisks\*`` 和 ``docs\[archive]``

| 步骤 | 负责人 | 备注 |
| ---- | ----- | ----- |
| 准备 | 文档 | 只替换句子，不要替换 ``docs/{{locale}}/index.mdx``。 |
| 验证 | QA | 检查“同步摘要”是否出现在报告中，并查看 [CLI 指南][cli-guide]。 |
| Publish | Ops | Upload ![Diagram](https://example.com/assets/flow(chart).png) after approval. |

1. 打开 `docs/index.mdx`。
2. 搜索 "同步摘要"。
3. 与上一版发布说明进行比较。

- 父项目
  - 嵌套笔记包含 `[Troubleshooting](https://example.com/docs/troubleshooting#common-errors)` 和 ``{{locale}}``

```bash
hyperlocalise run --group docs --dry-run
```

最终提醒：“同步摘要”必须在清单和报告中保持一致。

[cli-guide]: https://example.com/docs/cli(reference)
