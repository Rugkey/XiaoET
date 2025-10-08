#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""
小鹅通课程 m3u8 列表输出工具

使用方法:
    python main.py -c config.json              # 输出整个课程每个视频的 m3u8
    python main.py -c config.json --quality 720p
"""
import argparse
import os
import sys
import tempfile
import subprocess
from pathlib import Path

# 添加src目录到Python路径
sys.path.insert(0, str(Path(__file__).parent / 'src'))

from xiaoet_downloader import XiaoetConfig, logger
from xiaoet_downloader.tools.m3u8_lister import M3U8Lister


def main():
    """主函数"""
    parser = argparse.ArgumentParser(
        description='小鹅通课程 m3u8 列表输出工具',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
使用示例:
  python main.py --config config.json           # 输出整个课程每个视频的 m3u8
  python main.py -c config.json --quality 720p  # 指定清晰度
        """
    )
    
    parser.add_argument(
        '--config', '-c',
        default='config.json',
        help='配置文件路径 (默认: config.json)'
    )
    
    # 仅与 m3u8 获取相关的参数
    
    parser.add_argument(
        '--quality',
        choices=['auto','360p','480p','720p','1080p'],
        default='auto',
        help='导出指定清晰度的 m3u8（默认 auto 选择最佳）'
    )
    
    parser.add_argument(
        '--pipe-go',
        action='store_true',
        help='列出后直接调用 Go 批量下载器进行下载'
    )
    parser.add_argument(
        '--go-prefix',
        default='courseA',
        help='与 --pipe-go 搭配，Go 下载输出文件前缀 (默认: courseA)'
    )
    parser.add_argument(
        '--go-retries',
        type=int,
        default=3,
        help='与 --pipe-go 搭配，Go 每个请求的最大重试次数 (默认: 3)'
    )
    parser.add_argument(
        '--go-timeout',
        type=int,
        default=30,
        help='与 --pipe-go 搭配，Go 请求超时(秒) (默认: 30)'
    )
    
    
    parser.add_argument(
        '--verbose', '-v',
        action='store_true',
        help='显示详细日志'
    )
    
    args = parser.parse_args()
    
    # 设置日志级别
    if args.verbose:
        import logging
        logger.set_level(logging.DEBUG)
    
    try:
        # 加载配置
        if not os.path.exists(args.config):
            logger.error(f"配置文件不存在: {args.config}")
            logger.info("请创建配置文件，参考 config.json.example")
            return 1
        
        config = XiaoetConfig.from_file(args.config)
        # 直接列出 m3u8 地址
        logger.info("列出课程内所有视频的 m3u8 地址")
        lister = M3U8Lister(config)
        m3u8_list = lister.list_course_m3u8s(args.quality)
        for item in m3u8_list:
            logger.info(f"{item['title']} ({item['resource_id']}) [{item.get('quality','')}]: {item['m3u8']}")
        
        # 可选：管道到 Go 下载器
        if args.pipe_go and m3u8_list:
            # 写入临时 urls.txt（仅URL）
            with tempfile.NamedTemporaryFile(delete=False, suffix='.txt', mode='w', encoding='utf-8') as tf:
                for item in m3u8_list:
                    tf.write(item['m3u8'] + "\n")
                urls_path = tf.name
            logger.info(f"已生成临时 URL 列表: {urls_path}")
            
            # 规范化 Cookie
            cookie_value = (config.cookie or '').strip()
            if cookie_value.lower().startswith('cookie:'):
                cookie_value = cookie_value.split(':', 1)[1].strip()
            referer_value = f"https://{config.app_id}.h5.xiaoeknow.com/"
            
            # 计算 Go 项目路径
            go_dir = str((Path(__file__).parent / '..' / 'go-downloader').resolve())
            cmd = [
                'go','run','main.go',
                '-list', urls_path,
                '-prefix', args.go_prefix,
                '-cookie', cookie_value,
                '-referer', referer_value,
                '-retries', str(args.go_retries),
                '-timeout', str(args.go_timeout)
            ]
            logger.info(f"调用 Go 下载器: {' '.join(cmd)} (cwd={go_dir})")
            try:
                res = subprocess.run(cmd, cwd=go_dir, check=False)
                if res.returncode != 0:
                    logger.error(f"Go 下载器返回非零退出码: {res.returncode}")
                    return res.returncode
            finally:
                try:
                    os.remove(urls_path)
                except Exception:
                    pass
        
        return 0 if m3u8_list else 1
            
    except KeyboardInterrupt:
        logger.info("用户中断下载")
        return 130
    except Exception as e:
        logger.error(f"程序执行出错: {str(e)}")
        if args.verbose:
            import traceback
            logger.error(traceback.format_exc())
        return 1


if __name__ == '__main__':
    sys.exit(main())