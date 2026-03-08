---
title: "Danh sách kiểm tra khi phát hành"
description: "Kiểm tra và xác nhận các bản cập nhật tài liệu trước khi đồng bộ nội dung đã được dịch."
---

# Danh sách kiểm tra

Ship updates without breaking placeholders like `{{locale}}` or flags such as `--dry-run`.

Sử dụng [status reference](https://example.com/docs/status?tab=cli#dry-run) trước khi thực hiện thay đổi.

Reference links should also survive: [CLI guide][cli-guide] and ![Diagram](https://example.com/assets/flow(chart).png).

> Giữ các nhãn lặp lại ổn định.
> Giữ các nhãn lặp lại ổn định.
>
> Bảo toàn `MDPH_0_END` như văn bản thuần túy, không phải là token phân tích.

- Xem lại "Tóm tắt đồng bộ" trong terminal.
- Xác nhận xem các liên kết trong [Hướng dẫn khắc phục sự cố](https://example.com/docs/troubleshooting#common-errors) có còn hoạt động không.
- Không dịch `hyperlocalise run --group docs`
- Escape characters like `\*literal asterisks\*` and `docs\[archive]` carefully.

| Bước | Người chịu trách nhiệm | Ghi chú |
| ---- | ----- | ----- |
| Chuẩn bị | Tài liệu | Chỉ thay thế câu, không `docs/{{locale}}/index.mdx`. |
| Xác minh | Kiểm tra chất lượng | Kiểm tra xem "Tóm tắt đồng bộ" có xuất hiện trong báo cáo hay không và xem [hướng dẫn CLI][cli-guide]. |
| Publish | Ops | Upload ![Diagram](https://example.com/assets/flow(chart).png) after approval. |

1. Mở `docs/index.mdx`.
2. Tìm "Tóm tắt đồng bộ".
3. So sánh với bản ghi chú phiên bản trước.

- Mục cha
  - Ghi chú đệ quy với [Khắc phục sự cố](https://example.com/docs/troubleshooting#common-errors) và `{{locale}}`

```bash
hyperlocalise run --group docs --dry-run
```

Lưu ý cuối cùng: "Tóm tắt đồng bộ hóa" phải nhất quán giữa danh sách kiểm tra và báo cáo.

[cli-guide]: https://example.com/docs/cli(reference)
