# python-lister

仅输出课程内每个视频的 m3u8 地址，支持 --pipe-go 一键调用 Go 批量下载器。

用法:
```bash
python main.py -c config.json --quality 720p
python main.py -c config.json --quality 720p --pipe-go --go-prefix courseA
```

说明:
- 依赖本目录内的 `src/xiaoet_downloader` 组件（API、配置、日志、列表器）。
- 控制台输出，不写文件；若 `--pipe-go` 则生成临时列表并调用 Go 下载器。
