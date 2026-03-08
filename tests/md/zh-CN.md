---
title: "发布检查清单"
description: "在同步本地化内容之前，请先验证文档更新。"
---

# 发布检查清单

Ship updates without breaking placeholders like `{{locale}}` or flags such as `--dry-run`.

Use the [status reference](https://example.com/docs/status?tab=cli#dry-run) before you push changes.

参考链接也应保留：[CLI guide][cli-guide] 和 ![Diagram](https://example.com/assets/flow(chart).png).

> 保留重复的标签不变。
> 请保留重复标签的稳定性。
>
> Preserve `MDPH_0_END` as literal prose, not as a parser token.

- 在终端中查看"同步摘要"。
- Confirm links in [Troubleshooting](https://example.com/docs/troubleshooting#common-errors) stay intact.
- Do not translate `hyperlocalise run --group docs`.
- 小心地转义像 `\*literal asterisks\*` 和 `docs\[archive]` 这样的特殊字符。

| 步骤 | 负责人 | 备注 |
| ---- | ----- | ----- |
| 准备 | 文档 | 只替换句子，不要替换 `docs/{{locale}}/index.mdx`。 |
| 验证 | 质量保证 | 检查“同步摘要”是否出现在报告中，并查看 [CLI 指南][cli-guide]。 |
| 发布 | 运维 | 上传 ![Diagram](https://example.com/assets/flow(chart).png) 在批准后。 |

1. 打开 `docs/index.mdx`。
2. 搜索 "Sync summary"
3. 与之前的版本说明进行比较。

- 父项
  - 嵌套笔记：[故障排除](https://example.com/docs/troubleshooting#common-errors)和`{{locale}}`]

```bash
hyperlocalise run --group docs --dry-run
```

最后提醒：“同步摘要”必须在检查表和报告中保持一致。

[cli-guide]: https://example.com/docs/cli(reference)
