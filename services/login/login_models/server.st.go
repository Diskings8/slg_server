package login_models

const serverTable = "login_server"

// Server 区服表（落 common_db_0）
//
// ID 即 server_id，等于对应 game 节点的 instance（一个 game 进程 = 一个区服）。
type Server struct {
	ID         uint32 `gorm:"column:id;type:int(11);primary_key;not null;comment:server_id=game instance"` // server_id = game instance
	ServerName string `gorm:"column:server_name;type:varchar(64);not null;comment:区服名"`
	Status     int32  `gorm:"column:status;type:int(11);not null;default:0;comment:区服状态(0=开放 1=维护)"` // 0=开放 1=维护
	OpenTime   int64  `gorm:"column:open_time;type:bigint(20);not null;default:0;comment:开服时间"`
	CreatedAt  int64  `gorm:"column:created_at;type:bigint(20);not null;comment:创建时间"`
	UpdatedAt  int64  `gorm:"column:updated_at;type:bigint(20);not null;comment:更新时间"`
}

func (Server) TableName() string { return serverTable }
