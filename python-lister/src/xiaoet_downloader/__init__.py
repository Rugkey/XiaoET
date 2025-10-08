#!/usr/bin/env python
# -*- coding: utf-8 -*-

"""
小鹅通视频下载器

一个用于下载小鹅通课程视频的工具包
"""

__version__ = "2.0.0"
__author__ = "xiaoet-downloader"
__description__ = "小鹅通视频下载器"

# 延迟导入，避免在导入时就加载所有依赖
def _lazy_import():
    """延迟导入模块"""
    from .models.config import XiaoetConfig
    from .models.video import VideoResource, VideoMetadata, DownloadResult
    # 删除 manager 依赖
    from .utils.logger import logger
    
    return {
        'XiaoetConfig': XiaoetConfig,
        'VideoResource': VideoResource,
        'VideoMetadata': VideoMetadata,
        'DownloadResult': DownloadResult,
        'logger': logger
    }

# 使用__getattr__实现延迟导入
def __getattr__(name):
    if name in ['XiaoetConfig', 'VideoResource', 'VideoMetadata', 'DownloadResult', 'logger']:
        modules = _lazy_import()
        return modules[name]
    raise AttributeError(f"module '{__name__}' has no attribute '{name}'")

__all__ = [
    'XiaoetConfig',
    'VideoResource', 
    'VideoMetadata',
    'DownloadResult',
    'logger'
]