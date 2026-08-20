# -*- coding: utf-8 -*-
"""将 390 个 hero JSON 文件转换为配置表格式。

读取 C:/workspace/a4s/py/data/*.json，解析并生成：
  - hero_base.xlsx
  - hero_attr.xlsx
  - hero_method.xlsx
  - hero_method_slot.xlsx
  - hero_policy.xlsx
  - hero_group.xlsx
"""

import json
import os
from pathlib import Path
from openpyxl import Workbook


def load_hero_jsons(json_dir):
    """加载所有 hero JSON 文件。"""
    json_files = sorted(Path(json_dir).glob("*.json"))
    heroes = []
    for json_file in json_files:
        with open(json_file, 'r', encoding='utf-8') as f:
            data = json.load(f)
            heroes.append(data)
    print(f"加载了 {len(heroes)} 个英雄数据")
    return heroes


def extract_unique_methods(heroes):
    """提取所有唯一的战法。"""
    methods = {}
    for hero in heroes:
        for i in range(3):
            if i == 0:
                mid = hero.get('methodId')
                mname = hero.get('methodName')
                mdesc = hero.get('methodDesc')
            elif i == 1:
                mid = hero.get('methodId1')
                mname = hero.get('methodName1')
                mdesc = hero.get('methodDesc1')
            else:
                mid = hero.get('methodId2')
                mname = hero.get('methodName2')
                mdesc = hero.get('methodDesc2')

            if mid and mid != '' and mname:
                methods[mid] = {'method_id': mid, 'method_name': mname, 'method_desc': mdesc or ''}

    print(f"提取了 {len(methods)} 个唯一战法")
    return list(methods.values())


def extract_unique_policies(heroes):
    """提取所有唯一的政策。"""
    policies = {}
    for hero in heroes:
        pid = hero.get('policyId')
        pname = hero.get('policyName')
        pdesc = hero.get('policyDesc')
        if pid:
            policies[pid] = {'policy_id': pid, 'policy_name': pname or '', 'policy_desc': pdesc or ''}

    print(f"提取了 {len(policies)} 个唯一政策")
    return list(policies.values())


def extract_unique_groups(heroes):
    """提取所有唯一的羁绊组。"""
    groups = {}
    for hero in heroes:
        gid = hero.get('groupId')
        gname = hero.get('groupName')
        gdesc = hero.get('group')
        if gid and gid != '':
            groups[gid] = {'group_id': gid, 'group_name': gname or '', 'group_desc': gdesc or ''}

    print(f"提取了 {len(groups)} 个唯一羁绊组")
    return list(groups.values())


def build_hero_base(heroes):
    """构建 hero_base 表数据。"""
    rows = []
    for hero in heroes:
        row = {
            'conf_id': hero['id'],
            'name': hero.get('name', ''),
            'unique_name': hero.get('uniqueName', ''),
            'country': hero.get('country', ''),
            'sex': hero.get('sex', ''),
            'quality': hero.get('quality', ''),
            'icon_id': hero.get('iconId', 0),
            'troop_type': hero.get('type', ''),
            'troop_available': hero.get('type_availible', ''),
            'deploy_cost': float(hero.get('cost', 0)),
            'desc': hero.get('desc', ''),
            'policy_id': hero.get('policyId', 0),
            'group_id': hero.get('groupId', ''),
        }
        rows.append(row)
    return rows


def build_hero_attr(heroes):
    """构建 hero_attr 表数据。

    JSON 字段映射：
    - attack → base_attack
    - def → base_defense
    - ruse → base_intelligence
    - speed → base_movement
    - siege → base_relocation
    - distance → attack_range
    """
    rows = []
    for hero in heroes:
        row = {
            'conf_id': hero['id'],
            'base_attack': int(float(hero.get('attack', 0))),
            'base_defense': int(float(hero.get('def', 0))),
            'base_intelligence': int(float(hero.get('ruse', 0))),
            'base_movement': int(float(hero.get('speed', 0))),
            'base_relocation': int(float(hero.get('siege', 0))),
            'growth_attack': int(float(hero.get('attGrow', 0)) * 100),  # 成长率转为百分比
            'growth_defense': int(float(hero.get('defGrow', 0)) * 100),
            'growth_intelligence': int(float(hero.get('ruseGrow', 0)) * 100),
            'growth_movement': int(float(hero.get('speedGrow', 0)) * 100),
            'growth_relocation': int(float(hero.get('siegeGrow', 0)) * 100),
            'attack_range': hero.get('distance', 0),
        }
        rows.append(row)
    return rows


def build_hero_method_slots(heroes):
    """构建 hero_method_slot 表数据。"""
    rows = []
    for hero in heroes:
        hero_id = hero['id']

        # 槽位 0
        mid0 = hero.get('methodId')
        if mid0 and mid0 != '':
            rows.append({'hero_conf_id': hero_id, 'slot_index': 0, 'method_id': mid0})

        # 槽位 1
        mid1 = hero.get('methodId1')
        if mid1 and mid1 != '':
            rows.append({'hero_conf_id': hero_id, 'slot_index': 1, 'method_id': mid1})

        # 槽位 2
        mid2 = hero.get('methodId2')
        if mid2 and mid2 != '':
            rows.append({'hero_conf_id': hero_id, 'slot_index': 2, 'method_id': mid2})

    return rows


def write_xlsx(filename, headers, rows):
    """写入 xlsx 文件。

    第 1 行：中文标识名（headers 的 label）
    第 2 行起：数据
    """
    wb = Workbook()
    ws = wb.active

    # 写入表头（中文标识名）
    ws.append([h['label'] for h in headers])

    # 写入数据
    for row in rows:
        ws.append([row.get(h['name'], '') for h in headers])

    wb.save(filename)
    print(f"已生成: {filename} ({len(rows)} 行)")


def main():
    json_dir = r"C:\workspace\a4s\py\data"
    output_dir = Path(__file__).parent.parent / "excel"

    # 加载 JSON
    heroes = load_hero_jsons(json_dir)

    # 提取唯一数据
    methods = extract_unique_methods(heroes)
    policies = extract_unique_policies(heroes)
    groups = extract_unique_groups(heroes)

    # 构建表数据
    hero_base_rows = build_hero_base(heroes)
    hero_attr_rows = build_hero_attr(heroes)
    hero_method_slot_rows = build_hero_method_slots(heroes)

    # 定义表头（对应 gameconfig_schema.py 中的字段定义）
    hero_base_headers = [
        {'label': '英雄配置ID', 'name': 'conf_id'},
        {'label': '英雄名', 'name': 'name'},
        {'label': '唯一名', 'name': 'unique_name'},
        {'label': '阵营', 'name': 'country'},
        {'label': '性别', 'name': 'sex'},
        {'label': '品质', 'name': 'quality'},
        {'label': '图标ID', 'name': 'icon_id'},
        {'label': '主兵种', 'name': 'troop_type'},
        {'label': '可用兵种', 'name': 'troop_available'},
        {'label': '部署消耗', 'name': 'deploy_cost'},
        {'label': '描述', 'name': 'desc'},
        {'label': '政策ID', 'name': 'policy_id'},
        {'label': '羁绊组ID', 'name': 'group_id'},
    ]

    hero_attr_headers = [
        {'label': '英雄配置ID', 'name': 'conf_id'},
        {'label': '基础攻击', 'name': 'base_attack'},
        {'label': '基础防御', 'name': 'base_defense'},
        {'label': '基础智力', 'name': 'base_intelligence'},
        {'label': '基础移动', 'name': 'base_movement'},
        {'label': '基础拆迁', 'name': 'base_relocation'},
        {'label': '成长攻击', 'name': 'growth_attack'},
        {'label': '成长防御', 'name': 'growth_defense'},
        {'label': '成长智力', 'name': 'growth_intelligence'},
        {'label': '成长移动', 'name': 'growth_movement'},
        {'label': '成长拆迁', 'name': 'growth_relocation'},
        {'label': '攻击距离', 'name': 'attack_range'},
    ]

    hero_method_headers = [
        {'label': '战法ID', 'name': 'method_id'},
        {'label': '战法名', 'name': 'method_name'},
        {'label': '战法描述', 'name': 'method_desc'},
    ]

    hero_method_slot_headers = [
        {'label': '英雄配置ID', 'name': 'hero_conf_id'},
        {'label': '槽位索引', 'name': 'slot_index'},
        {'label': '战法ID', 'name': 'method_id'},
    ]

    hero_policy_headers = [
        {'label': '政策ID', 'name': 'policy_id'},
        {'label': '政策名', 'name': 'policy_name'},
        {'label': '政策描述', 'name': 'policy_desc'},
    ]

    hero_group_headers = [
        {'label': '羁绊组ID', 'name': 'group_id'},
        {'label': '羁绊名', 'name': 'group_name'},
        {'label': '羁绊描述', 'name': 'group_desc'},
    ]

    # 写入 xlsx
    write_xlsx(output_dir / "hero_base.xlsx", hero_base_headers, hero_base_rows)
    write_xlsx(output_dir / "hero_attr.xlsx", hero_attr_headers, hero_attr_rows)
    write_xlsx(output_dir / "hero_method.xlsx", hero_method_headers, methods)
    write_xlsx(output_dir / "hero_method_slot.xlsx", hero_method_slot_headers, hero_method_slot_rows)
    write_xlsx(output_dir / "hero_policy.xlsx", hero_policy_headers, policies)
    write_xlsx(output_dir / "hero_group.xlsx", hero_group_headers, groups)

    print("\n转换完成！")


if __name__ == '__main__':
    main()
