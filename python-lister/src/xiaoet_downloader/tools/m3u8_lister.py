#!/usr/bin/env python
# -*- coding: utf-8 -*-

from typing import List, Dict

from ..models.config import XiaoetConfig
from ..api.client import XiaoetAPIClient
from ..utils.logger import logger


class M3U8Lister:
    """仅列出课程视频 m3u8 地址的工具类"""

    def __init__(self, config: XiaoetConfig):
        self.config = config
        self.api_client = XiaoetAPIClient(config)

    def list_course_m3u8s(self, desired_quality: str = "auto") -> List[Dict[str, str]]:
        """列出课程内所有视频的 m3u8 地址
        desired_quality: "auto" 或 360p/480p/720p/1080p
        """
        items: List[Dict[str, str]] = []
        try:
            navigation_info = self.api_client.get_micro_navigation_info()
            user_id = navigation_info.get('user_id') or \
                      navigation_info.get('user', {}).get('user_id') or \
                      navigation_info.get('user_info', {}).get('user_id')
            if not user_id:
                logger.error("无法获取用户ID")
                return items

            resource_items = self.api_client.get_column_items(self.config.product_id)
            if not resource_items:
                logger.warning("未找到课程资源")
                return items

            logger.info(f"找到 {len(resource_items)} 个资源，开始获取 m3u8")
            for index, (resource_id, resource_title) in enumerate(resource_items):
                try:
                    if not resource_id.startswith('v_'):
                        logger.info(f"跳过非视频资源: {resource_title}")
                        continue

                    # 直接用 APIClient 获取 play_sign -> play_url
                    video_details = self.api_client.get_video_detail_info(resource_id)
                    play_sign = video_details.get('play_sign')
                    if not play_sign:
                        logger.warning(f"无法获取播放标识: {resource_title}")
                        continue

                    play_list_dict = self.api_client.get_play_url(user_id, play_sign)
                    # 选择清晰度
                    quality_map = {
                        '1080p': '1080p_hls',
                        '720p': '720p_hls',
                        '480p': '480p_hls',
                        '360p': '360p_hls'
                    }
                    play_url = None
                    quality = None
                    if desired_quality and desired_quality != 'auto':
                        key = quality_map.get(desired_quality)
                        if key and key in play_list_dict and play_list_dict.get(key, {}).get('play_url'):
                            play_url = play_list_dict.get(key, {}).get('play_url')
                            quality = key
                    if not play_url:
                        play_url, quality = self.api_client.get_best_quality_url(play_list_dict)
                    if play_url:
                        items.append({
                            'resource_id': resource_id,
                            'title': resource_title,
                            'm3u8': play_url,
                            'quality': quality
                        })
                        logger.info(f"[{index+1}/{len(resource_items)}] {quality}: {resource_title}")
                    else:
                        logger.warning(f"[{index+1}/{len(resource_items)}] 无法获取 m3u8: {resource_title}")
                except Exception as e:
                    logger.error(f"获取视频 {resource_title} 的 m3u8 时出错: {str(e)}")
        except Exception as e:
            logger.error(f"列出 m3u8 时发生错误: {str(e)}")
        return items


