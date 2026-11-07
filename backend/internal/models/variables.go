package models

// Variable 定义工程内的数值变量；玩家当前值只存在于播放器存档。
type Variable struct {
	ID           uint   `gorm:"primaryKey;comment:变量主键" json:"id"`                                       // 变量 ID
	ProjectID    uint   `gorm:"not null;uniqueIndex:idx_variable_name;comment:所属工程 ID" json:"projectId"` // 所属工程
	Name         string `gorm:"not null;uniqueIndex:idx_variable_name;comment:工程内唯一变量名称" json:"name"`    // 显示名称
	InitialValue int64  `gorm:"not null;default:0;comment:新游戏初始整数值" json:"initialValue"`                 // 初始值
	ResetOnLoop  bool   `gorm:"not null;default:false;comment:新一轮循环时是否恢复初始值" json:"resetOnLoop"`         // 循环重置策略
}

// PortCondition 是端口出现的一个比较条件。
type PortCondition struct {
	ID         uint   `gorm:"primaryKey;comment:条件主键" json:"id"`                 // 条件 ID
	PortID     uint   `gorm:"not null;index;comment:所属出端口 ID" json:"portId"`     // 出端口
	VariableID uint   `gorm:"not null;index;comment:检查的变量 ID" json:"variableId"` // 变量
	Operator   string `gorm:"not null;comment:比较运算符" json:"operator"`            // >= <= == != > <
	Value      int64  `gorm:"not null;comment:比较阈值" json:"value"`                // 阈值
}

// PortEffect 表示经过端口时对变量执行一次增减。
type PortEffect struct {
	ID         uint  `gorm:"primaryKey;comment:效果主键" json:"id"`                                               // 效果 ID
	PortID     uint  `gorm:"not null;uniqueIndex:idx_port_variable;comment:所属出端口 ID" json:"portId"`           // 出端口
	VariableID uint  `gorm:"not null;index;uniqueIndex:idx_port_variable;comment:修改的变量 ID" json:"variableId"` // 变量
	Delta      int64 `gorm:"not null;comment:正数增加负数减少" json:"delta"`                                          // 变化量
}
