---
title: "Kiểm tra trước khi phát hành"
description: "Vui lòng xác minh các bản cập nhật tài liệu trước khi đồng bộ hóa nội dung đã dịch."
---

# Kiểm tra trước khi phát hành

Ship updates without breaking placeholders like `{{locale}}` or flags such as `--dry-run`.

Use the [status reference](https://example.com/docs/status?tab=cli#dry-run) before you push changes.

Reference links should also survive: [CLI guide][cli-guide] and ![Diagram](https://example.com/assets/flow(chart).png).

> Duy trì các nhãn lặp lại ổn định.
> Duy trì các nhãn lặp lại ổn định.
>
> Preserve `MDPH_0_END` as literal prose, not as a parser token.

- Xem lại "Tóm tắt so sánh" trong terminal.
- Xác nhận rằng các liên kết trong [Khắc phục sự cố](https://example.com/docs/troubleshooting#common-errors) vẫn giữ nguyên.
- Do not translate `hyperlocalise run --group docs`.
- Tránh các ký tự thoát như `\*literal asterisks\*` và `docs\[archive]` một cách cẩn thận.

| Bước | Người thực hiện | Ghi chú |
| ---- | ----- | ----- |
| Chuẩn bị | Tài liệu | Chỉ thay thế câu, không thay đổi "`docs/{{locale}}/index.mdx`". |
| Xác minh | Kiểm tra chất lượng | Kiểm tra xem "Tóm tắt đồng bộ" có xuất hiện trong báo cáo hay không và xem xét [hướng dẫn CLI][cli-guide]. |
| Publish | Ops | Upload ![Diagram](https://example.com/assets/flow(chart).png) after approval. |

1. Mở [HLMDPH_857FCB6CC4A4_0]
2. Tìm kiếm "Sync summary".
3. So sánh với bản ghi chú phiên bản trước.

- Mục cha
  - Nested note with [Troubleshooting](https://example.com/docs/troubleshooting#common-errors) and `{{locale}}`

```bash
hyperlocalise run --group docs --dry-run
```

Lưu ý cuối cùng: "Tóm tắt đồng bộ" phải nhất quán trên danh sách kiểm tra và báo cáo.

[cli-guide]: https://example.com/docs/cli(reference)
