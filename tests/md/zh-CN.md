---
title: "发布检查清单"
description: "在同步本地化内容之前，请验证文档更新。"
---

# 发布检查清单

Ship updates without breaking placeholders like `{{locale}}` or flags such as `--dry-run`.

Use the [status reference](https://example.com/docs/status?tab=cli#dry-run) before you push changes.

参考链接也应保持：[CLI指南][cli-guide] 和 ![图表](https://example.com/assets/flow(chart).png)

> 保持重复的标签不变。
> 保持重复的标签不变。
>
> 保留 `MDPH_0_END` 作为字面文本，而不是作为解析器标记。

- 在终端中查看“同步摘要”。
- Confirm links in [Troubleshooting](https://example.com/docs/troubleshooting#common-errors) stay intact.
- 请勿翻译 `hyperlocalise run --group docs`。
- 小心地转义像 `\*literal asterisks\*` 和 `docs\[archive]` 这样的特殊字符。

| 步骤 | 负责人 | 备注 |
| ---- | ----- | ----- |
| 准备 | 文档 | 仅替换句子，而不是 `docs/{{locale}}/index.mdx`。 |
| 验证 | QA | 检查“同步摘要”是否出现在报告中，并查看 [CLI 指南][cli-guide]。 |
| 发布 | 运维 | 审核通过后，上传 ![图表](https://example.com/assets/flow(chart).png)。 |

1. 打开 `docs/index.mdx`。
2. 搜索“同步摘要”。
3. 与之前的版本说明进行比较。

- 父项
  - 嵌套笔记中包含[故障排除](https://example.com/docs/troubleshooting#common-errors)和`{{locale}}`]

```bash
hyperlocalise run --group docs --dry-run
```

最后提醒：“同步摘要”必须在清单和报告中保持一致。

[cli-guide]: https://example.com/docs/cli(reference)
