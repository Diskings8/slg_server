package env_confs

import (
	"fmt"
)

type mysql struct {
	User         string `yaml:"user"`
	Pass         string `yaml:"pass"`
	Addr         string `yaml:"addr"`
	Port         string `yaml:"port"`
	DbNamePrefix string `yaml:"db_name_prefix"`
	Params       string `yaml:"params"`
}

type redis struct {
	Addr     string `yaml:"addr"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

func (r redis) Dsn() string {
	return fmt.Sprintf("%s:%s", r.Addr, r.Port)
}

type etcd struct {
	Addr string `yaml:"addr"`
	Port string `yaml:"port"`
}

func (e etcd) Dsn() string {
	return fmt.Sprintf("%s:%s", e.Addr, e.Port)
}

type snowflake struct {
	DatacenterID int64 `yaml:"datacenter_id"`
	WorkerID     int64 `yaml:"worker_id"`
}

type gateway struct {
	Addr    string `yaml:"addr"`
	TcpPort string `yaml:"tcp_port"`
	RpcPort string `yaml:"rpc_port"`
}

func (gw gateway) TcpDsn() string {
	return fmt.Sprintf("%s:%s", gw.Addr, gw.TcpPort)
}

func (gw gateway) RpcDsn() string {
	return fmt.Sprintf("%s:%s", gw.Addr, gw.RpcPort)
}

type server struct {
	Addr string `yaml:"addr"`
	Port string `yaml:"port"`
}

func (s server) Dsn() string {
	return fmt.Sprintf("%s:%s", s.Addr, s.Port)
}

func (ms *mysql) Dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", ms.User, ms.Pass, ms.Addr, ms.Port, ms.DbNamePrefix, ms.Params)
}

type Config struct {
	MysqlCommon mysql     `yaml:"mysql_common"`
	MysqlGame   mysql     `yaml:"mysql_game"`
	Redis       redis     `yaml:"redis"`
	Snowflake   snowflake `yaml:"snowflake"`
	Etcd        etcd      `yaml:"etcd"`
	GateWay     gateway   `yaml:"gateway"`

	GameServer   server `yaml:"game_server"`
	BattleServer server `yaml:"battle_server"`
}
