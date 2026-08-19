# -*- coding: utf-8 -*-
"""悬空引用修复数据：为现有 13 表迁移补齐缺失的关键行（一次性）。

gen_gameconfig_xlsx.py 在读取现有 json 后、写入 xlsx 前合并这些行。

- ITEM_OVERRIDES：
  review_level 奖励引用 item 1002/1003 但 item.json 缺失 → 补齐后满足 item.Validate。
  - 1002 金币礼包·小：效果=加二级货币(100002=金币) 500
  - 1003 经验书·小：效果=加英雄经验 50
- BUILDING_OVERRIDES：
  city.proto BuildingType 含 RoleMilitary(103) 但 building.json 缺失 → 与 barracks(104) 同构补齐。
  102 RoleBranchCity 有意不配：归 worldmap OverlayEvent（代码注释明示）。
"""

# 金币礼包·小 / 经验书·小（item 效果枚举：0=None,1=AddHeroExp,2=AddCurrency,3=AddItem）
ITEM_OVERRIDES = [
    {"conf_id": 1002, "effect": {"type": 2, "target": 100002, "value": 500}},
    {"conf_id": 1003, "effect": {"type": 1, "target": 0, "value": 50}},
]

# 军事建筑（RoleMilitary, BuildingType=103）：2x2，与 barracks(104) 同构的曲线
BUILDING_OVERRIDES = [
    {"type": 103, "name": "military", "footprint": 4, "max_level": 10,
     "build_time_ux": 120,
     "build_cost": [
         {"item_id": 100004, "item_type": 3, "count": 200},
         {"item_id": 100005, "item_type": 3, "count": 200},
         {"item_id": 100003, "item_type": 3, "count": 150},
         {"item_id": 100006, "item_type": 3, "count": 100}],
     "upgrade_cost_base": [
         {"item_id": 100004, "item_type": 3, "count": 300},
         {"item_id": 100005, "item_type": 3, "count": 300},
         {"item_id": 100003, "item_type": 3, "count": 200},
         {"item_id": 100006, "item_type": 3, "count": 150}],
     "upgrade_cost_growth": 1.5, "upgrade_time_growth": 1.3},
]
