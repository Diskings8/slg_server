---
name: gen-entity
description: 从 game_models 定义生成 game_entitys 层代码(st.go/func.go/func.gen.go/db.go)并注册到 Role
---

# gen-entity — 正向生成 Game Entity

## 用法

```
/gen-entity <model文件路径>
```

例如 `/gen-entity services/game/game_models/model.hero.go`

我会读取该 model 的 struct 定义，然后按以下步骤生成完整的 entity 层代码。

---

## 执行步骤

### 1. 读取 Model

- 读取指定的 `model.<name>.go` 文件
- 提取 struct 名、字段名和类型、TableName、约束信息(uniqueIndex 等)

### 2. 推导模块命名

| 输入 | 输出 |
|------|------|
| 文件 `model.<name>.go` | 目录 `game_entitys/game_roles/<module_name>/` |
| struct `XxxYyy` | Entity `XxxYyy`, Collection `XxxYyy` + `s` |
| 文件前缀: `model.<a>.<b>.go` → `<b>`, `model.<a>.go` → `<a>` |

> 包名从目录名来。struct 名 + `s` 作为 Collection 名，如果 struct 名以 `s` 结尾则加 `es`。

### 3. 生成 4 个文件

#### `<prefix>.st.go`
- Entity 结构体: 内嵌 `*game_models.<ModelName>`
- Collection 结构体: `List`, `Mem`(hashmaps.Map), `RoleID`
- 包含所有必要的 import

#### `<prefix>.func.go`
- `New<CollectionName>(roleID) *<CollectionName>` — 构造函数
- `(cs *<CollectionName>) Init()` — 遍历 List 填充 Mem
- `(cs *<CollectionName>) Copy(src *<CollectionName>)` — JSON 拷贝 + Init
- `New<EntityName>(one *game_models.<ModelName>) *<EntityName>` — 实体构造函数
- `Format2Pb()` — 如果 model 引用了 pb 类型则生成，否则不加

使用 `util_jsons` 做 marshal/unmarshal，`loggers` 打错误日志。

#### `<prefix>.func.gen.go`
- 为 model 的每个字段生成 `Get<Field>() <Type>` / `Set<Field>(v <Type>)`，包括 ModelBase(ID/CreatedAt/UpdatedAt)
- import model 中引用的所有 pb 包

#### `<prefix>.db.go`
- `Init(writeDB)` — AutoMigrate
- `DBCreate`, `DBDelete`, `DBSave`, `DBGet` — 标准 CRUD
- DBGet 要处理 `gorm.ErrRecordNotFound`

### 4. 注册到 Role

修改 4 个文件：

#### `game.role.st.go`
- Role struct 添加字段 `FieldName *<module_pkg>.<CollectionName>`
- `New()` 中添加 `r.FieldName = <module_pkg>.New<CollectionName>(roleID)`
- 添加 Getter 方法(副本懒拷贝模式)

#### `game.role.copy.go`
- `Reset()` 中添加 `r.FieldName = nil`

#### `game.role.poller.go`
- `Init()` 中添加 `<module_pkg>.Init(writeDB)`

---

## 检查清单

生成完后自检:
- [ ] 4 个文件都已创建且编译通过
- [ ] `Role.New()` 实例化了新模块
- [ ] `Role.Reset()` 清空了新模块指针
- [ ] `Role.GetXXX()` Getter 实现了副本懒拷贝
- [ ] `poller.go Init()` 注册了 AutoMigrate
- [ ] 整体 `go build ./...` 通过
