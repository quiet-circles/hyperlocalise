---
title: "Danh sách kiểm tra khi phát hành"
description: "Kiểm tra và xác nhận các bản cập nhật tài liệu trước khi đồng bộ hóa nội dung đã dịch."
---

# Danh sách kiểm tra

Ship updates without breaking placeholders like `{{locale}}` or flags such as `--dry-run`.

Hãy sử dụng [status reference](https://example.com/docs/status?tab=cli#dry-run) trước khi thực hiện thay đổi.

Các liên kết tham khảo cũng nên tồn tại: [Hướng dẫn dòng lệnh][cli-guide] và ![Đồ họa](https://example.com/assets/flow(chart).png).

> Giữ các nhãn lặp lại ổn định.
> Giữ các nhãn lặp lại ổn định.
>
> Preserve `MDPH_0_END` as literal prose, not as a parser token.

- Xem lại "Tóm tắt đồng bộ" trong cửa sổ lệnh.
- Confirm links in [Troubleshooting](https://example.com/docs/troubleshooting#common-errors) stay intact.
- Không dịch `hyperlocalise run --group docs`.
- Escape characters like `\*literal asterisks\*` and `docs\[archive]` carefully.

| Bước | Người chịu trách nhiệm | Ghi chú |
| ---- | ----- | ----- |
| Chuẩn bị | Tài liệu | Chỉ thay thế câu, không `docs/{{locale}}/index.mdx`. |
| Xác minh | Kiểm tra chất lượng | Kiểm tra xem "Tóm tắt đồng bộ" có xuất hiện trong báo cáo hay không và xem [hướng dẫn CLI][cli-guide]. |
| Publish | Ops | Upload ![Diagram](https://example.com/assets/flow(chart).png) after approval. |

1. Mở `docs/index.mdx`.
2. Tìm kiếm "Tóm tắt đồng bộ".
3. So sánh với bản ghi chú của phiên bản trước.

- Mục cha
  - Ghi chú đệ quy với [Khắc phục sự cố](https://example.com/docs/troubleshooting#common-errors) và `{{locale}}`

```bash
hyperlocalise run --group docs --dry-run
```

Lưu ý cuối: "Tóm tắt đồng bộ hóa" phải nhất quán trong danh sách kiểm tra và báo cáo.

[cli-guide]: https://example.com/docs/cli(reference)
