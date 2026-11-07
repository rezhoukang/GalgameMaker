package models

// Variable 定义工程内的数值变量；玩家当前值只存在于播放器存档。
type Variable struct {
	ID           uint   `gorm:"primaryKey;comment:变量主键" json:"id"`                                       // 变量 ID
	ProjectID    uint   `gorm:"not null;uniqueIndex:idx_variable_name;comment:所属工程 ID" json:"projectId"` // 所属工程
	Name         string `gorm:"not null;uniqueIndex:idx_variable_name;comment:工程内唯一变量名称" json:"name"`    // 显示名称
	InitialValue int64  `gorm:"not null;default:0;comment:新游戏初始整数值" json:"initialValue"`                 // 初始值
}

// PortCondition 是 ports.conditions JSON 数组中的一条出现条件，不独立建表。
type PortCondition struct {
	VariableID uint   `json:"variableId"` // 检查的变量 ID
	Operator   string `json:"operator"`   // >= <= == != > <
	Value      int64  `json:"value"`      // 比较阈值
}

// PortEffect 是 ports.effects JSON 数组中的一条变量操作，不独立建表。
type PortEffect struct {
	VariableID uint  `json:"variableId"` // 修改的变量 ID
	Set        int   `json:"set"`        // 0 按 delta 增减，1 直接设为 delta
	Delta      int64 `json:"delta"`      // set=0 时表示增减量，set=1 时表示目标值（0 即清空）
}
