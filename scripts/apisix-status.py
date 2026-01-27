#!/usr/bin/env python3
"""美化显示 APISIX 配置状态"""

import json
import sys
import urllib.request
import os
from pathlib import Path

def load_env():
    """加载项目根目录的 .env 文件"""
    # 查找项目根目录的 .env 文件
    script_dir = Path(__file__).parent
    project_root = script_dir.parent
    env_file = project_root / '.env'
    
    if env_file.exists():
        with open(env_file) as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith('#') and '=' in line:
                    key, _, value = line.partition('=')
                    key = key.strip()
                    value = value.strip()
                    # 只在环境变量未设置时才使用 .env 中的值
                    if key not in os.environ:
                        os.environ[key] = value

load_env()

ADMIN_URL = os.environ.get('APISIX_ADMIN_URL', 'http://localhost:9180')
ADMIN_KEY = os.environ.get('APISIX_ADMIN_KEY', 'edd1c9f034335f136f87ad84b625c8f1')

def fetch(endpoint):
    """获取 APISIX Admin API 数据"""
    try:
        req = urllib.request.Request(
            f"{ADMIN_URL}/apisix/admin/{endpoint}",
            headers={"X-API-KEY": ADMIN_KEY}
        )
        with urllib.request.urlopen(req, timeout=5) as resp:
            return json.loads(resp.read().decode())
    except Exception:
        return None

def print_header():
    print()
    print("╔════════════════════════════════════════════════════════════════════════════╗")
    print("║                        📊 APISIX 当前配置状态                              ║")
    print("╚════════════════════════════════════════════════════════════════════════════╝")
    print()

def print_routes():
    print("┌─────────────────────────────────────────────────────────────────────────────┐")
    print("│ 🛣️  路由 (Routes)                                                           │")
    print("├─────────────────────────────────────────────────────────────────────────────┤")
    
    data = fetch("routes")
    if data is None:
        print("│  ⚠️  无法连接 APISIX Admin API                                              │")
    else:
        routes = data.get('list', [])
        if not routes:
            print("│  (无路由配置)                                                              │")
        else:
            for r in routes:
                v = r.get('value', {})
                uri = v.get('uri', 'N/A')
                methods = ','.join(v.get('methods', ['*']))
                name = v.get('name', v.get('id', 'N/A'))
                status = '🟢' if v.get('status', 1) == 1 else '🔴'
                print(f"│  {status} {methods:<8} {uri:<35} [{name}]")
    
    print("└─────────────────────────────────────────────────────────────────────────────┘")
    print()

def print_consumers():
    print("┌─────────────────────────────────────────────────────────────────────────────┐")
    print("│ 👤 消费者 (Consumers)                                                       │")
    print("├─────────────────────────────────────────────────────────────────────────────┤")
    
    data = fetch("consumers")
    if data is None:
        print("│  ⚠️  无法连接 APISIX Admin API                                              │")
    else:
        consumers = data.get('list', [])
        if not consumers:
            print("│  (无消费者配置)                                                            │")
        else:
            for c in consumers:
                v = c.get('value', {})
                username = v.get('username', 'N/A')
                plugins = list(v.get('plugins', {}).keys())
                plugin_str = ', '.join(plugins[:3]) + ('...' if len(plugins) > 3 else '') if plugins else '无插件'
                print(f"│  • {username:<20} 插件: {plugin_str}")
    
    print("└─────────────────────────────────────────────────────────────────────────────┘")
    print()

def print_global_rules():
    print("┌─────────────────────────────────────────────────────────────────────────────┐")
    print("│ 🌐 全局规则 (Global Rules)                                                  │")
    print("├─────────────────────────────────────────────────────────────────────────────┤")
    
    data = fetch("global_rules")
    if data is None:
        print("│  ⚠️  无法连接 APISIX Admin API                                              │")
    else:
        rules = data.get('list', [])
        if not rules:
            print("│  (无全局规则)                                                              │")
        else:
            for r in rules:
                v = r.get('value', {})
                rule_id = v.get('id', 'N/A')
                plugins = list(v.get('plugins', {}).keys())
                plugin_str = ', '.join(plugins[:4]) + ('...' if len(plugins) > 4 else '') if plugins else '无插件'
                print(f"│  • 规则 #{rule_id:<15} 插件: {plugin_str}")
    
    print("└─────────────────────────────────────────────────────────────────────────────┘")
    print()

if __name__ == '__main__':
    print_header()
    print_routes()
    print_consumers()
    print_global_rules()
