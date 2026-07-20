package roles

import (
	"database/sql/driver"

	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/utils/util_jsons"
)

func (d *Data) Save(isDelete bool) error {
	if isDelete {
		return nil
	}
	return d.DBSave()
}

// Value gorm使用
func (d *Data) Value() (driver.Value, error) { return util_jsons.Marshal(d) }

// Scan gorm使用
func (d *Data) Scan(input any) error { return util_jsons.Unmarshal(input.([]byte), d) }

// DBCreate 增
func (d *Data) DBCreate() error {
	return dbconn.GetWriteDbConn().Save(d).Error()
}

// DBDelete 删
func (d *Data) DBDelete() error {
	return dbconn.GetWriteDbConn().Where("role_id = ?", d.RoleID).Delete(d).Error()
}

// DBSave 改
func (d *Data) DBSave() error {
	return dbconn.GetWriteDbConn().Save(d).Error()
}

// DBGet 查
func (d *Data) DBGet() error {
	return dbconn.GetReadDbConn().Where("role_id = ? ", d.RoleID).Take(d).Error()
}
