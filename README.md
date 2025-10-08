# m3u8-suite 花了一天结合AI做了好久，哈哈，先发出来1.0版本，目前我这已经更新到3.0，打包成windows.exe可执行文件了，求支持。。。

一个将“小鹅通课程 m3u8 列表导出”和“m3u8 批量下载”结合的最小套件：
- python-lister：列出课程每个视频的 m3u8（支持指定清晰度）。
- go-downloader：使用原生 HTTP 下载并用 ffmpeg 合并，支持 Cookie/Referer、重试、IV 解析。

## 目录结构
- python-lister/
  - main.py（支持 --pipe-go 一键列出+下载）
  - src/xiaoet_downloader（API、配置、日志与列表器）
- go-downloader/
  - main.go（-list/-prefix/-cookie/-referer/-retries/-timeout）

## 环境要求
- Python 3.9+
- Go 1.20+
- ffmpeg（在 PATH 中）

## 快速开始
1) 准备配置文件 `python-lister/config.json`（参考同目录 `config.json.example`）：
```json
{
  "app_id": "your_app_id",
  "cookie": "Cookie: xxx=...; yyy=...",
  "product_id": "p_xxx",
  "download_dir": "download"
}
```
2) 仅列出 m3u8：
```bash
python python-lister/main.py -c python-lister/config.json --quality 720p
```
3) 一键列出+下载（调用 Go 下载器）：
```bash
python python-lister/main.py -c python-lister/config.json --quality 720p --pipe-go --go-prefix courseA
```
等同内部执行：
```bash
go run go-downloader/main.go -list <临时urls.txt> -prefix courseA -cookie "<cookie>" -referer "https://<app_id>.h5.xiaoeknow.com/" -retries 3 -timeout 30
```

## 说明与建议
- Cookie 可直接粘贴整行 `Cookie: ...`，程序会自动剥离前缀。
- `--quality` 支持 auto/360p/480p/720p/1080p；auto 模式选择最佳清晰度。
- 若需要 JSON/CSV 的机器可读输出，可在 Python 侧扩展（当前仅控制台输出）。
- Go 下载器已使用原生 HTTP，不依赖 wget；仍依赖系统 ffmpeg 进行合并。

## 故障排查
- 未找到课程资源：检查 `app_id/product_id` 是否来自同一个课程页面，Cookie 是否新鲜且对应登录态。
- Go 下载器报错：检查 Go/ffmpeg 是否在 PATH 中；网络是否可访问；可调整 `-retries/-timeout`。

