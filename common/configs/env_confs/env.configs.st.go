package env_confs

import "fmt"

// ================================================================
// 格式中立配置结构体
//
// YAML / TOML 共用同一套字段标签，无需手动映射。
// ================================================================

// ─── 数据库 ──────────────────────────────────────────────────────

// DBInstance 数据库实例（对应 db.common / db.game）
type DBInstance struct {
	User     string `yaml:"user" toml:"user"`
	Password string `yaml:"password" toml:"password"`
	Host     string `yaml:"host" toml:"host"`
	Port     int    `yaml:"port" toml:"port"`
	DBName   string `yaml:"db_name" toml:"db_name"`
	Params   string `yaml:"params" toml:"params"`
}

func (d DBInstance) Dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", d.User, d.Password, d.Host, d.Port, d.DBName, d.Params)
}

// DBConfig 数据库配置
type DBConfig struct {
	Common DBInstance `yaml:"common" toml:"common"`
	Game   DBInstance `yaml:"game" toml:"game"`
}

// ─── 缓存 ────────────────────────────────────────────────────────

// CacheConfig 缓存配置
type CacheConfig struct {
	Prefix   string   `yaml:"prefix" toml:"prefix"`
	Address  []string `yaml:"address" toml:"address"`
	Password string   `yaml:"password" toml:"password"`
	DB       int      `yaml:"db" toml:"db"`
	NodeType string   `yaml:"node_type" toml:"node_type"`
}

func (c CacheConfig) GetNodeType() string { return c.NodeType }
func (c CacheConfig) GetPrefix() string   { return c.Prefix }

// ─── 服务发现 ────────────────────────────────────────────────────

// EtcdConfig 服务发现配置
type EtcdConfig struct {
	Address []string `yaml:"address" toml:"address"`
}

func (e EtcdConfig) Dsn() string {
	if len(e.Address) > 0 {
		return e.Address[0]
	}
	return ""
}

// ─── ID 生成器 ──────────────────────────────────────────────────

// SnowflakeConfig ID 生成器配置
type SnowflakeConfig struct {
	DatacenterID int64 `yaml:"datacenter_id" toml:"datacenter_id"`
	WorkerID     int64 `yaml:"worker_id" toml:"worker_id"`
}

// ─── 网关 ────────────────────────────────────────────────────────

// GatewayConfig 网关节点配置
type GatewayConfig struct {
	TCPAddr string `yaml:"tcp_addr" toml:"tcp_addr"`
	RPCAddr string `yaml:"rpc_addr" toml:"rpc_addr"`
}

func (g GatewayConfig) TcpDsn() string { return g.TCPAddr }
func (g GatewayConfig) RpcDsn() string { return g.RPCAddr }

// ─── 通用节点 ────────────────────────────────────────────────────

// NodeConfig 通用服务节点（game / battle / worldmap）
type NodeConfig struct {
	Addr string `yaml:"addr" toml:"addr"`
}

func (n NodeConfig) Dsn() string { return n.Addr }

// ─── 其它 ────────────────────────────────────────────────────────

// CommonConfig 通用环境信息
type CommonConfig struct {
	ID        int    `yaml:"id" toml:"id"`
	Name      string `yaml:"name" toml:"name"`
	IsDevelop bool   `yaml:"is_develop" toml:"is_develop"`
}

// GameConfConfig 游戏配置表路径
type GameConfConfig struct {
	ConfigPath string `yaml:"config_path" toml:"config_path"`
}

// ─── 顶级 ────────────────────────────────────────────────────────

// Config 全局环境配置。
//
// YAML / TOML 文件结构与此结构体完全一致，可直接反序列化，无格式偏见。
type Config struct {
	Common       CommonConfig    `yaml:"common" toml:"common"`
	DB           DBConfig        `yaml:"db" toml:"db"`
	Cache        CacheConfig     `yaml:"cache" toml:"cache"`
	Etcd         EtcdConfig      `yaml:"etcd" toml:"etcd"`
	Snowflake    SnowflakeConfig `yaml:"snowflake" toml:"snowflake"`
	Gateway      GatewayConfig   `yaml:"gateway" toml:"gateway"`
	Game         NodeConfig      `yaml:"game" toml:"game"`
	Battle       NodeConfig      `yaml:"battle" toml:"battle"`
	BattleRecord NodeConfig      `yaml:"battle_record" toml:"battle_record"`
	Worldmap     NodeConfig      `yaml:"worldmap" toml:"worldmap"`
	Login        NodeConfig      `yaml:"login" toml:"login"`
	GameConf     GameConfConfig  `yaml:"game_conf" toml:"game_conf"`

	// 程序运行时设置，不从配置文件读取
	NodeType string
}

func (c *Config) SetNodeType(nodeType string) { c.NodeType = nodeType }
func (c *Config) GetNodeType() string         { return c.NodeType }
