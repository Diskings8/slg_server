# -*- coding: utf-8 -*-
"""配置表结构 spec（单一事实源）。

本文件定义 a4s 配置管线的全部表结构：
  - TABLES   : 24 张数据表（进 Index.xlsx，对应运行时 pb.Table 的 repeated 字段）
  - STRUCTS  : 共享结构体（game_attribute 行组，不进 Index，对应 pb 独立 message）
  - ENUMS    : 枚举（进 game_enumeration.xlsx，成员按声明顺序自动编号 0,1,2…）

gen_gameconfig_xlsx.py 据此生成：
  1. Index.xlsx（表注册）
  2. game_attribute.xlsx（字段注册，中文标识名 → 英文字段名 → 类型）
  3. game_enumeration.xlsx（枚举定义）
  4. 24 张数据表 xlsx（第 1 行中文标识名表头，第 2 行起数据）

字段类型取值：uint32/int32/int64/uint64/bool/string/float，或 STRUCTS/ENUMS 中的类型名。
数组字段 array=True（game_attribute 数组切割="|"；数据表用 "|" 连接多值）。
索引字段 index=True（game_attribute 索引="1"）。
"""


class Field(object):
    def __init__(self, label, name, ftype, array=False, index=False, comment=""):
        self.label = label            # 中文标识名（xlsx 表头 + game_attribute 标识名 + 结构体单元格键）
        self.name = name              # 英文字段名（proto 字段名 / json key）
        self.type = ftype             # 字段类型（基本类型 / 结构体名 / 枚举名）
        self.array = array            # 是否 repeated（数组切割="|"）
        self.index = index            # 是否主键（索引="1"）
        self.comment = comment        # 注释（proto 注释 + xlsx 注释列）

    def to_dict(self):
        return dict(label=self.label, name=self.name, type=self.type,
                    array=self.array, index=self.index, comment=self.comment)


# ── 共享结构体（对象类型=结构体名 的行组定义字段） ──
STRUCTS = {
    # 资源/货币消耗：item_id + item_type + count（对应 common_declarations.ItemUse）
    "cost": [
        Field("道具", "item_id", "int64", comment="道具配置ID"),
        Field("类型", "item_type", "int32", comment="道具类型（0=普通/1=一级货币/2=二级货币/3=资源）"),
        Field("数量", "count", "int64", comment="数量"),
    ],
    # 奖励：item_id + count（对应 ReviewReward / 英雄卡收集）
    "reward": [
        Field("道具", "item_id", "int64", comment="道具配置ID（英雄卡时=英雄配置ID）"),
        Field("数量", "count", "int64", comment="数量"),
    ],
    # 等级→数值断点（对应 building.LevelNum）
    "level_num": [
        Field("等级", "level", "uint32", comment="等级"),
        Field("数值", "num", "uint32", comment="该等级数值"),
    ],
}

# ── 枚举（game_enumeration；成员 = (枚举全名, 备注)） ──
ENUMS = {
    # 技能类型（skill.SkillType）
    "skilltype": [
        ("skilltype_none", "无"),
        ("skilltype_active", "主动技能（行动时先手释放）"),
        ("skilltype_pursuit", "追击技能（普攻后按概率触发）"),
        ("skilltype_passive", "被动技能（buff，预留）"),
    ],
    # 技能目标选择（skill.TargetType）
    "targettype": [
        ("targettype_none", "无"),
        ("targettype_random", "攻击范围内随机单个"),
        ("targettype_front", "前排（距己方大营最近）"),
        ("targettype_base", "大营"),
    ],
    # 技能效果类型（skill.EffectType）
    "effecttype": [
        ("effecttype_none", "无"),
        ("effecttype_phys_damage", "物理伤害：攻击 vs 防御，收敛公式"),
        ("effecttype_magic_damage", "法术伤害：双方智力，收敛公式"),
        ("effecttype_recover", "恢复伤兵"),
    ],
    # 道具效果类型（item.ItemEffectType）
    "itemeffecttype": [
        ("itemeffecttype_none", "无效果（仅消耗）"),
        ("itemeffecttype_add_hero_exp", "加英雄经验"),
        ("itemeffecttype_add_currency", "加货币（target=货币配置ID）"),
        ("itemeffecttype_add_item", "资源包，加道具（target=道具配置ID）"),
    ],
    # 资源地产出方式（resource.ResourceType）
    "resourcetype": [
        ("resourcetype_mixed", "混合型：全类型各产 amount（lv1）"),
        ("resourcetype_dual", "双资源：主资源 + 随机次级资源（lv2）"),
        ("resourcetype_single", "单项：只产主资源（lv3+）"),
    ],
}

# ── 数据表（kind: single=单行 / multi=多行，index 字段为行主键） ──
TABLES = [
    # battle（战斗规则，single 行）
    dict(name="battle", kind="single", comment="战斗规则",
         fields=[
             Field("回合数", "rounds", "uint32", comment="战斗回合数"),
             Field("首回合受伤比例", "injury_rate_start", "uint32", comment="第1回合受伤比例(%)"),
             Field("受伤比例递减", "injury_rate_decay", "uint32", comment="每回合受伤比例递减(%)"),
             Field("结算死亡比例", "settlement_dead_rate", "uint32", comment="结算阶段每回合伤兵转死亡比例(%)"),
             Field("物理收敛系数", "phys_converge", "uint32", comment="物理伤害收敛系数(%)"),
             Field("法术收敛系数", "magic_converge", "uint32", comment="法术伤害收敛系数(%)"),
             Field("战斗经验系数", "battle_exp_coeff", "uint32", comment="战斗经验系数"),
         ]),

    # formation（编队，single 行）
    dict(name="formation", kind="single", comment="编队配置",
         fields=[
             Field("最大槽位数", "max_slots", "uint32", comment="编队最大槽位数"),
         ]),

    # troop（兵种，single 行，transform_cost 为 repeated cost）
    dict(name="troop", kind="single", comment="兵种配置",
         fields=[
             Field("转化等级", "transform_level", "uint32", comment="兵种转化所需等级"),
             Field("默认兵种ID", "default_troop_id", "int32", comment="默认兵种配置ID"),
             Field("解锁道具", "unlock_item_conf", "int32", comment="扩展兵种解锁道具配置ID"),
             Field("转化消耗", "transform_cost", "cost", array=True, comment="转化消耗（资源混搭）"),
         ]),

    # hero（英雄系统标量，single 行；经验/英雄属性见 hero_exp/hero_attr）
    dict(name="hero", kind="single", comment="英雄系统配置",
         fields=[
             Field("英雄等级上限", "max_level", "uint32", comment="英雄等级上限"),
             Field("每10级自由点", "free_point_per_10l", "uint32", comment="每10级获得的自由属性点"),
             Field("星级上限", "max_star_stage", "int32", comment="星级上限"),
             Field("每星自由点", "star_point_per", "uint32", comment="每升1星发放的自由属性点"),
             Field("觉醒等级", "awaken_level", "uint32", comment="觉醒等级门槛"),
             Field("觉醒消耗", "awaken_cost", "cost", array=True, comment="觉醒消耗（资源混搭）"),
         ]),

    # hero_exp（升级经验曲线，multi，level 主键）
    dict(name="hero_exp", kind="multi", comment="英雄升级经验",
         fields=[
             Field("等级", "level", "uint32", index=True, comment="英雄等级（1~max_level）"),
             Field("升级经验", "exp", "uint32", comment="从该级升到下一级所需经验"),
         ]),

    # hero_attr（每英雄属性，multi，conf_id 主键；base/growth 摊平）
    dict(name="hero_attr", kind="multi", comment="英雄属性",
         fields=[
             Field("英雄配置ID", "conf_id", "int32", index=True, comment="英雄配置ID"),
             Field("基础攻击", "base_attack", "uint32", comment="基础攻击（lv1）"),
             Field("基础防御", "base_defense", "uint32", comment="基础防御（lv1）"),
             Field("基础智力", "base_intelligence", "uint32", comment="基础智力（lv1）"),
             Field("基础移动", "base_movement", "uint32", comment="基础移动（lv1）"),
             Field("基础拆迁", "base_relocation", "uint32", comment="基础拆迁（lv1）"),
             Field("成长攻击", "growth_attack", "uint32", comment="每级攻击成长"),
             Field("成长防御", "growth_defense", "uint32", comment="每级防御成长"),
             Field("成长智力", "growth_intelligence", "uint32", comment="每级智力成长"),
             Field("成长移动", "growth_movement", "uint32", comment="每级移动成长"),
             Field("成长拆迁", "growth_relocation", "uint32", comment="每级拆迁成长"),
             Field("攻击距离", "attack_range", "uint32", comment="攻击距离"),
         ]),

    # item（道具，multi，conf_id 主键；effect 摊平）
    dict(name="item", kind="multi", comment="道具配置",
         fields=[
             Field("道具配置ID", "conf_id", "int32", index=True, comment="道具配置ID"),
             Field("效果类型", "effect_type", "itemeffecttype", comment="道具效果类型"),
             Field("效果目标", "effect_target", "int32", comment="目标配置ID（货币/道具）"),
             Field("效果数值", "effect_value", "int64", comment="单个道具的效果数值"),
         ]),

    # exchange（货币兑换，multi，from_id 主键）
    dict(name="exchange", kind="multi", comment="货币兑换",
         fields=[
             Field("来源货币ID", "from_id", "int32", index=True, comment="来源货币配置ID（一级货币）"),
             Field("来源货币类型", "from_type", "int32", comment="来源货币类型（1=一级/2=二级）"),
             Field("目标货币ID", "to_id", "int32", comment="目标货币配置ID（二级货币）"),
             Field("目标货币类型", "to_type", "int32", comment="目标货币类型（1=一级/2=二级）"),
             Field("消耗数量", "from_count", "int64", comment="消耗来源货币数量（每组）"),
             Field("获得数量", "to_count", "int64", comment="获得目标货币数量（每组）"),
         ]),

    # resource（资源地产量，multi，level 主键）
    dict(name="resource", kind="multi", comment="资源地产量",
         fields=[
             Field("等级", "level", "int32", index=True, comment="资源地等级 1~9"),
             Field("产出方式", "res_type", "resourcetype", comment="产出方式（type 为 Go/proto 保留字故用 res_type）"),
             Field("产量", "amount", "int32", comment="混合/单项产量"),
             Field("主资源产量", "primary_amount", "int32", comment="双资源主产量"),
             Field("次级资源产量", "secondary_amount", "int32", comment="双资源次级产量"),
         ]),

    # gacha_pool（抽卡池，multi，pool_id 主键）
    dict(name="gacha_pool", kind="multi", comment="抽卡池",
         fields=[
             Field("抽卡池ID", "pool_id", "int32", index=True, comment="抽卡池ID"),
             Field("池名", "name", "string", comment="抽卡池名称"),
             Field("抽卡券道具", "ticket_conf_id", "int32", comment="抽卡券道具配置ID（0=无券模式）"),
             Field("单抽券数", "single_ticket", "int64", comment="单抽消耗券数"),
             Field("单抽金币", "single_gold", "int64", comment="单抽消耗金币数"),
             Field("十连券数", "ten_ticket", "int64", comment="十连消耗券数"),
             Field("十连金币", "ten_gold", "int64", comment="十连消耗金币数"),
             Field("每日免费", "free_daily", "bool", comment="每日免费窗口各1次"),
             Field("半价", "half_price", "bool", comment="半价窗口各1次"),
             Field("保底次数", "guarantee_times", "int32", comment="累抽N次必出高稀有度"),
             Field("保底组", "guarantee_group_id", "int32", comment="保底命中时走的掉落组"),
             Field("首抽保底组", "first_drop_group_id", "int32", comment="首抽保底组（0=无）"),
             Field("心愿英雄", "wish_heros", "int32", array=True, comment="心愿可选英雄配置ID集合"),
             Field("心愿阈值", "wish_times", "int32", comment="心愿进度阈值"),
         ]),

    # gacha_drop_group（掉落组，multi，复合键 (pool_id,group_id)；tabtoy 索引须唯一故不标 index）
    dict(name="gacha_drop_group", kind="multi", comment="抽卡掉落组",
         fields=[
             Field("抽卡池ID", "pool_id", "int32", comment="归属抽卡池"),
             Field("掉落组ID", "group_id", "int32", comment="组ID（档位）"),
             Field("组权重", "weight", "int32", comment="非保底抽取时的组权重"),
         ]),

    # gacha_drop_item（掉落条目，multi）
    dict(name="gacha_drop_item", kind="multi", comment="抽卡掉落条目",
         fields=[
             Field("抽卡池ID", "pool_id", "int32", comment="归属抽卡池"),
             Field("掉落组ID", "group_id", "int32", comment="归属掉落组"),
             Field("奖励配置ID", "reward_conf_id", "int32", comment="英雄或道具配置ID"),
             Field("是否英雄", "is_hero", "bool", comment="true=英雄卡；false=道具"),
             Field("数量", "count", "int32", comment="产出数量（英雄卡恒1）"),
             Field("权重", "weight", "int32", comment="组内权重"),
             Field("保底重置", "guarantee_reset", "bool", comment="命中后保底计数归零"),
         ]),

    # guard（守军标量，single 行）
    dict(name="guard", kind="single", comment="守军配置",
         fields=[
             Field("最高开发等级", "max_develop_level", "int32", comment="地块可开发的最高等级上限"),
         ]),

    # guard_config（守军等级，multi，level 主键；槽位见 guard_slot）
    dict(name="guard_config", kind="multi", comment="守军等级配置",
         fields=[
             Field("地块等级", "level", "int32", index=True, comment="地块等级（守军查表索引）"),
         ]),

    # guard_slot（守军槽位，multi）
    dict(name="guard_slot", kind="multi", comment="守军槽位",
         fields=[
             Field("地块等级", "level", "int32", comment="归属地块等级"),
             Field("英雄配置ID", "hero_conf_id", "int32", comment="守军英雄配置ID"),
             Field("兵力", "soldier_num", "uint32", comment="固定兵力"),
         ]),

    # soldier（兵力标量，single 行；断点见 soldier_hero_cap / soldier_barrack_bonus）
    dict(name="soldier", kind="single", comment="兵力配置",
         fields=[
             Field("默认兵力", "default_soldier_num", "uint32", comment="英雄上阵默认兵力"),
         ]),

    # soldier_hero_cap（英雄等级兵力断点，multi，level 主键）
    dict(name="soldier_hero_cap", kind="multi", comment="英雄等级兵力断点",
         fields=[
             Field("英雄等级", "level", "int32", index=True, comment="英雄等级断点"),
             Field("基础兵力", "soldier_num", "uint32", comment="该等级及以上的基础兵力"),
         ]),

    # soldier_barrack_bonus（兵营等级加成断点，multi，level 主键）
    dict(name="soldier_barrack_bonus", kind="multi", comment="兵营等级兵力加成断点",
         fields=[
             Field("兵营等级", "level", "int32", index=True, comment="兵营等级断点"),
             Field("累计加成", "bonus", "uint32", comment="该等级的累计兵力加成"),
         ]),

    # review（审查标量，single 行；等级见 review_level）
    dict(name="review", kind="single", comment="审查玩法配置",
         fields=[
             Field("每日次数", "daily_chances", "int32", comment="每天获得审查次数"),
             Field("最大累积", "max_chances", "int32", comment="最多累积次数"),
             Field("每次任务数", "tasks_per_review", "int32", comment="每次审查生成任务数"),
             Field("最低经验", "exp_per_review_min", "int32", comment="每次审查最低经验"),
             Field("最高经验", "exp_per_review_max", "int32", comment="每次审查最高经验"),
             Field("赛季天数", "season_days", "int32", comment="赛季天数"),
             Field("升级赠次数", "level_up_bonus_chances", "int32", comment="前N级每升1级送1次审查次数"),
         ]),

    # review_level（审查等级，multi，level 主键；rewards 为 repeated reward）
    dict(name="review_level", kind="multi", comment="审查等级",
         fields=[
             Field("审查等级", "level", "int32", index=True, comment="审查等级"),
             Field("升级经验", "exp_required", "int32", comment="升到该等级所需累计经验"),
             Field("任务奖励", "rewards", "reward", array=True, comment="该等级任务奖励（道具）"),
         ]),

    # skill（技能，multi，conf_id 主键；upgrade_cost 为单 cost）
    dict(name="skill", kind="multi", comment="技能配置",
         fields=[
             Field("技能配置ID", "conf_id", "int32", index=True, comment="技能配置ID"),
             Field("等级上限", "max_level", "int32", comment="等级上限"),
             Field("装配上限", "use_limit", "int32", comment="可装配次数上限"),
             Field("升级消耗", "upgrade_cost", "cost", comment="单次升级消耗"),
             Field("技能类型", "skill_type", "skilltype", comment="技能类型"),
             Field("目标选择", "target_type", "targettype", comment="目标选择"),
             Field("效果类型", "effect_type", "effecttype", comment="效果类型"),
             Field("伤害系数", "damage_coeff", "uint32", comment="技能伤害系数(%)，0=按基础攻/智算"),
             Field("收敛系数", "converge_coeff", "uint32", comment="技能收敛系数(%)，0=用战斗规则全局系数"),
             Field("触发概率", "trigger_rate", "uint32", comment="追击触发概率(%)，仅追击技能有效"),
         ]),

    # skill_collection（技能收藏，multi，skill_conf_id 主键；need_heroes 为 repeated reward）
    dict(name="skill_collection", kind="multi", comment="技能收藏",
         fields=[
             Field("技能配置ID", "skill_conf_id", "int32", index=True, comment="技能配置ID"),
             Field("所需英雄卡", "need_heroes", "reward", array=True, comment="所需英雄卡（ItemID=英雄配置ID）"),
         ]),

    # skill_setting（技能槽位标量，single 行）
    dict(name="skill_setting", kind="single", comment="技能槽位配置",
         fields=[
             Field("默认槽位", "slot_default", "int32", comment="index0 默认技能槽（英雄自带）"),
             Field("装配槽位下限", "slot_equip_min", "int32", comment="可装配槽位起始"),
             Field("装配槽位上限", "slot_equip_max", "int32", comment="可装配槽位上限"),
             Field("槽位1解锁等级", "slot1_unlock_lv", "uint32", comment="槽位1（index1）解锁所需英雄等级"),
             Field("槽位2解锁等级", "slot2_unlock_lv", "uint32", comment="槽位2（index2）解锁所需英雄等级"),
             Field("拆卸返还比例", "unequip_refund", "int32", comment="拆卸返还比例（每升1级返1/N）"),
         ]),

    # building（建筑，multi，btype 主键；cost/queue_nums 为数组）
    dict(name="building", kind="multi", comment="建筑配置",
         fields=[
             Field("建筑类型", "btype", "int32", index=True, comment="建筑类型（BuildingType；type 为保留字故用 btype）"),
             Field("名称", "name", "string", comment="建筑名称"),
             Field("占地", "footprint", "int32", comment="占地格子数（4=2x2/9=3x3）"),
             Field("等级上限", "max_level", "uint32", comment="等级上限"),
             Field("建造耗时", "build_time_ux", "int64", comment="建造耗时(秒)；0=即时"),
             Field("建造消耗", "build_cost", "cost", array=True, comment="建造消耗（可空=免费）"),
             Field("升级基础消耗", "upgrade_cost_base", "cost", array=True, comment="1→2 升级消耗基数"),
             Field("升级消耗成长", "upgrade_cost_growth", "double", comment="升级消耗成长（每级乘数）"),
             Field("升级耗时成长", "upgrade_time_growth", "double", comment="升级耗时成长（每级乘数）"),
             Field("队列数断点", "queue_nums", "level_num", array=True, comment="校场：等级→队列数断点"),
             Field("每级防御", "defense_per_level", "uint32", comment="城墙：每级防御加成"),
             Field("每级容量", "cap_per_level", "uint64", comment="仓库：每级资源存量上限"),
             Field("产出道具", "produce_item", "int32", comment="资源建筑：产出资源配置ID"),
             Field("每小时产出", "produce_per_hour_l", "int64", comment="资源建筑：每级每小时产出"),
         ]),
]
