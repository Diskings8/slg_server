# -*- coding: utf-8 -*-
"""验证 gameconfig.json 中的 hero 数据。"""

import json
from pathlib import Path

json_path = Path(__file__).parent.parent / "json" / "gameconfig.json"

with open(json_path, 'r', encoding='utf-8') as f:
    data = json.load(f)

print("=" * 50)
print("Hero 数据导入验证")
print("=" * 50)

tables = ['hero_base', 'hero_attr', 'hero_method', 'hero_method_slot', 'hero_policy', 'hero_group']

for table in tables:
    count = len(data.get(table, []))
    print(f"{table:20s}: {count:4d} 条记录")

print("\n" + "=" * 50)
print("hero_base 第一条记录样本:")
print("=" * 50)

if data.get('hero_base'):
    hero = data['hero_base'][0]
    for key, value in hero.items():
        if isinstance(value, str) and len(value) > 50:
            value = value[:50] + "..."
        print(f"  {key:20s}: {value}")

print("\n" + "=" * 50)
print("hero_method 前 3 条:")
print("=" * 50)

if data.get('hero_method'):
    for i, method in enumerate(data['hero_method'][:3], 1):
        print(f"{i}. ID={method['method_id']}, 名称={method['method_name']}")
        print(f"   描述={method['method_desc'][:60]}...")

print("\n验证完成！")
