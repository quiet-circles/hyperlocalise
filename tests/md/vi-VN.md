---
title: "Danh sách kiểm tra phát hành"
description: "Hãy xác minh các bản cập nhật tài liệu trước khi đồng bộ hóa nội dung đã dịch."
---

# Danh sách kiểm tra phát hành

Ship updates without breaking placeholders like `{{locale}}` or flags such as `--dry-run`.

Use the [status reference](https://example.com/docs/status?tab=cli#dry-run) before you push changes.

Reference links should also survive: [CLI guide][cli-guide] and ![Diagram](https://example.com/assets/flow(chart).png).

> Duy trì các nhãn lặp lại một cách ổn định.
> Giữ các nhãn lặp lại ổn định
>
> Preserve `MDPH_0_END` as literal prose, not as a parser token.

- Xem lại "Tóm tắt đồng bộ" trong terminal.
- Xác nhận các liên kết trong [Hướng dẫn khắc phục sự cố](https://example.com/docs/troubleshooting#common-errors) vẫn giữ nguyên.
- Do not translate `hyperlocalise run --group docs`.
- Các ký tự thoát như `\*literal asterisks\*` và `docs\[archive]` cần được xử lý cẩn thận.

| Bước | Người chịu trách nhiệm | Ghi chú |
| ---- | ----- | ----- |
| Chuẩn bị | Tài liệu | Chỉ thay thế câu, không thay đổi ``docs/{{locale}}/index.mdx``.
| Xác minh | Kiểm tra chất lượng | Kiểm tra xem "Tóm tắt đồng bộ" có xuất hiện trong báo cáo hay không và xem xét [Hướng dẫn CLI][cli-guide]. |
| Publish | Ops | Upload ![Diagram](https://example.com/assets/flow(chart).png) after approval. |

1. Mở `docs/index.mdx`.
2. Tìm kiếm "Tóm tắt đồng bộ".
3. So sánh với bản ghi chú phát hành trước.

- Mục cha
  - Ghi chú lồng nhau với [Khắc phục sự cố](https://example.com/docs/troubleshooting#common-errors) và`{{locale}}`

```bash
hyperlocalise run --group docs --dry-run
```

Lưu ý cuối cùng: "Tóm tắt đồng bộ" phải nhất quán trên danh sách kiểm tra và báo cáo.

[cli-guide]: https://example.com/docs/cli(reference)
