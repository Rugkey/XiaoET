# go-downloader

基于 Go 的 m3u8 批量下载器（原生 HTTP，支持 Cookie/Referer、重试、IV 解析）。

用法:
```bash
go run main.go -list urls.txt -prefix courseA -cookie "k=v" -referer "https://example.com" -retries 3 -timeout 30
```

说明:
- 使用系统 ffmpeg 合并为 `merge.ts`，随后移动到上级目录。
- ts 下载与 key 获取均用原生 HTTP，不依赖 wget。
