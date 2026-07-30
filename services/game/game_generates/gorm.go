package main

import (
	"fmt"
	"os"

	configs "server.slg.com/common/configs"

	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

const genEnv = "dev"

// GenDB 从数据库表生成 model + query 代码
func GenDB() {
	// 1. 加载配置
	configPath := fmt.Sprintf("../../api/yaml_conf/slg.%s.yaml", genEnv)
	configs.LoadYamlConf(configPath)

	dsn := configs.GetConf().DB.Game.Dsn()
	if dsn == "" {
		fmt.Fprintf(os.Stderr, "❌ mysql_game DSN 为空，请检查 %s\n", configPath)
		os.Exit(1)
	}

	// 2. 直连 MySQL（gen 需要 *gorm.DB，不走 dbconn 包装层）
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"❌ 连接数据库失败: %v\n\n"+
				"  请确认:\n"+
				"    1. MySQL 已启动 (127.0.0.1:3306)\n"+
				"    2. 配置正确: %s\n"+
				"    3. 数据库 %s 存在\n\n"+
				"  你也可以修改 game_generates/gorm.go 顶部的 genEnv 变量\n"+
				"  来切换配置文件 (dev/prod/test)\n",
			err, configPath, configs.GetConf().DB.Game.DBName)
		os.Exit(1)
	}

	// 3. 初始化 gen
	g := gen.NewGenerator(gen.Config{
		OutPath:           "../game_querys", // DAO 查询代码输出
		ModelPkgPath:      "game_models",    // 模型 struct 输出
		Mode:              gen.WithDefaultQuery | gen.WithQueryInterface | gen.WithoutContext,
		WithUnitTest:      false,
		FieldNullable:     true,
		FieldSignable:     true,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
	})
	g.UseDB(db)

	// 4. 注册自定义类型的导入路径
	g.WithImportPkgPath(
		"server.slg.com/api/protocol/pb/pb_gameconfig",
		"server.slg.com/api/protocol/pb/pb_roledata",
	)

	// 5. 注册表映射
	applyAllModels(g)

	// 6. 执行生成
	g.Execute()
}

// applyAllModels 注册所有表的生成规则
func applyAllModels(g *gen.Generator) {
	// ==================== 角色相关 ====================

	g.ApplyBasic(
		// 角色主表
		g.GenerateModel("role",
			gen.FieldGORMTag("id", func(tag field.GormTag) field.GormTag {
				return tag.Set("autoIncrement", "false")
			}),
		),

		// 角色属性
		g.GenerateModel("role_attr",
			gen.FieldGORMTag("role_id", func(tag field.GormTag) field.GormTag {
				return tag.Set("autoIncrement", "false")
			}),
			gen.FieldType("lang", "pb_gameconfig.LangType"),
			gen.FieldGenType("lang", "Int32"),
			gen.FieldType("country", "pb_gameconfig.State"),
			gen.FieldType("create_country", "pb_gameconfig.State"),
			gen.FieldType("head_customs", "[4]*pb_roledata.HeadCustom"),
			JsonSerializer("head_customs"),
		),

		// 角色英雄
		g.GenerateModel("role_heroes",
			gen.FieldGORMTag("role_id", func(tag field.GormTag) field.GormTag {
				return tag.Set("autoIncrement", "false")
			}),
		),

		// 角色道具
		g.GenerateModel("role_items",
			gen.FieldGORMTag("role_id", func(tag field.GormTag) field.GormTag {
				return tag.Set("autoIncrement", "false")
			}),
		),

		// 角色建筑
		g.GenerateModel("role_builds",
			gen.FieldGORMTag("role_id", func(tag field.GormTag) field.GormTag {
				return tag.Set("autoIncrement", "false")
			}),
		),

		// 角色装备
		g.GenerateModel("role_equips",
			gen.FieldGORMTag("role_id", func(tag field.GormTag) field.GormTag {
				return tag.Set("autoIncrement", "false")
			}),
		),

		// 角色队伍
		g.GenerateModel("role_teams",
			gen.FieldGORMTag("role_id", func(tag field.GormTag) field.GormTag {
				return tag.Set("autoIncrement", "false")
			}),
		),

		// 角色联盟
		g.GenerateModel("role_union",
			gen.FieldGORMTag("role_id", func(tag field.GormTag) field.GormTag {
				return tag.Set("autoIncrement", "false")
			}),
		),

		// 角色任务
		g.GenerateModel("role_task",
			gen.FieldGORMTag("role_id", func(tag field.GormTag) field.GormTag {
				return tag.Set("autoIncrement", "false")
			}),
		),
	)

	// ==================== 联盟相关 ====================

	g.ApplyBasic(
		g.GenerateModel("unions"),
	)

	// ==================== 系统相关 ====================

	g.ApplyBasic(
		g.GenerateModel("system_activity"),
	)
}

// JsonSerializer 为指定列添加 JSON 序列化标签
func JsonSerializer(columnName string) gen.ModelOpt {
	return gen.FieldGORMTag(columnName, func(tag field.GormTag) field.GormTag {
		tag.Set("serializer", "json")
		return tag
	})
}
