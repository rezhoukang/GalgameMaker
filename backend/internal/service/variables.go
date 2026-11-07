package service

import (
	"errors"
	"fmt"
	"galgame-maker/internal/models"
	"gorm.io/gorm"
	"strings"
)

const maxStoryValue int64 = 9007199254740991 // JavaScript 可精确表示的最大整数。
func validStoryValue(v int64) bool           { return v >= -maxStoryValue && v <= maxStoryValue }

func projectVariables(db *gorm.DB, projectID uint) ([]models.Variable, error) {
	vars := []models.Variable{}
	err := db.Where("project_id = ?", projectID).Order("id").Find(&vars).Error
	return vars, err
}

func loadPortRules(db *gorm.DB, ports []models.Port) error {
	ids := []uint{}
	byID := map[uint]*models.Port{}
	for i := range ports {
		p := &ports[i]
		p.Conditions = []models.PortCondition{}
		p.Effects = []models.PortEffect{}
		ids = append(ids, p.ID)
		byID[p.ID] = p
	}
	if len(ids) == 0 {
		return nil
	}
	var conditions []models.PortCondition
	var effects []models.PortEffect
	if err := db.Where("port_id IN ?", ids).Order("id").Find(&conditions).Error; err != nil {
		return err
	}
	if err := db.Where("port_id IN ?", ids).Order("id").Find(&effects).Error; err != nil {
		return err
	}
	for _, c := range conditions {
		byID[c.PortID].Conditions = append(byID[c.PortID].Conditions, c)
	}
	for _, e := range effects {
		byID[e.PortID].Effects = append(byID[e.PortID].Effects, e)
	}
	return nil
}

// PortRules 作为一个整体保存，避免只保存了部分条件或效果。
type PortRules struct {
	ConditionMode string                 `json:"conditionMode"` // all / any
	ResetLoop     bool                   `json:"resetLoop"`     // 先重置本轮变量
	Conditions    []models.PortCondition `json:"conditions"`    // 条件列表
	Effects       []models.PortEffect    `json:"effects"`       // 变量变化列表
}

func validateRules(vars []models.Variable, mode string, conditions []models.PortCondition, effects []models.PortEffect) error {
	if mode != "all" && mode != "any" {
		return errors.New("条件组合方式必须为 all 或 any")
	}
	available := map[uint]bool{}
	for _, v := range vars {
		available[v.ID] = true
	}
	for _, c := range conditions {
		if !available[c.VariableID] {
			return errors.New("条件引用的变量不属于当前工程")
		}
		if !validStoryValue(c.Value) {
			return errors.New("条件值必须为安全整数")
		}
		switch c.Operator {
		case ">=", "<=", "==", "!=", ">", "<":
		default:
			return errors.New("不支持的比较方式")
		}
	}
	seen := map[uint]bool{}
	for _, e := range effects {
		if !available[e.VariableID] {
			return errors.New("效果引用的变量不属于当前工程")
		}
		if seen[e.VariableID] {
			return errors.New("同一端口的每个变量只配置一条变化")
		}
		if !validStoryValue(e.Delta) {
			return errors.New("变化量必须为安全整数")
		}
		seen[e.VariableID] = true
	}
	return nil
}

func savePortRules(tx *gorm.DB, p *models.Port, rules PortRules) error {
	var scene models.Node
	if err := activeScenes(tx).First(&scene, p.NodeID).Error; err != nil {
		return err
	}
	vars, err := projectVariables(tx, scene.ProjectID)
	if err != nil {
		return err
	}
	if err = validateRules(vars, rules.ConditionMode, rules.Conditions, rules.Effects); err != nil {
		return err
	}
	if err = deletePortRules(tx, []uint{p.ID}); err != nil {
		return err
	}
	p.ConditionMode = rules.ConditionMode
	p.ResetLoop = rules.ResetLoop
	p.Conditions = []models.PortCondition{}
	p.Effects = []models.PortEffect{}
	for _, c := range rules.Conditions {
		c.ID = 0
		c.PortID = p.ID
		if err = tx.Create(&c).Error; err != nil {
			return err
		}
		p.Conditions = append(p.Conditions, c)
	}
	for _, e := range rules.Effects {
		e.ID = 0
		e.PortID = p.ID
		if err = tx.Create(&e).Error; err != nil {
			return err
		}
		p.Effects = append(p.Effects, e)
	}
	return nil
}

func deletePortRules(tx *gorm.DB, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Where("port_id IN ?", ids).Delete(&models.PortCondition{}).Error; err != nil {
		return err
	}
	return tx.Where("port_id IN ?", ids).Delete(&models.PortEffect{}).Error
}

// SaveVariables 替换变量定义，保留 ID；正在被条件或效果引用的变量不可删除。
func (s *NodeService) SaveVariables(projectID uint, vars []models.Variable) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var project models.Project
		if err := tx.First(&project, projectID).Error; err != nil {
			return errors.New("工程不存在")
		}
		current, err := projectVariables(tx, projectID)
		if err != nil {
			return err
		}
		available := map[uint]bool{}
		kept := map[uint]bool{}
		names := map[string]bool{}
		for _, v := range current {
			available[v.ID] = true
		}
		for i := range vars {
			v := &vars[i]
			v.Name = strings.TrimSpace(v.Name)
			v.ProjectID = projectID
			if v.Name == "" || names[v.Name] {
				return errors.New("变量名称不能为空且不可重复")
			}
			names[v.Name] = true
			if !validStoryValue(v.InitialValue) {
				return errors.New("变量初始值必须为安全整数")
			}
			if v.ID != 0 {
				if !available[v.ID] || kept[v.ID] {
					return errors.New("变量 ID 无效或不属于当前工程")
				}
				kept[v.ID] = true
			}
		}
		for _, v := range current {
			if !kept[v.ID] {
				var count int64
				if err = tx.Model(&models.PortCondition{}).Where("variable_id = ?", v.ID).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return fmt.Errorf("变量「%s」仍被端口条件使用", v.Name)
				}
				if err = tx.Model(&models.PortEffect{}).Where("variable_id = ?", v.ID).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return fmt.Errorf("变量「%s」仍被端口效果使用", v.Name)
				}
			}
		}
		// 事务内重建定义以支持名称互换，明确 ID 的记录先插入，新增记录最后插入。
		if err = tx.Where("project_id = ?", projectID).Delete(&models.Variable{}).Error; err != nil {
			return err
		}
		for _, fresh := range []bool{false, true} {
			for _, v := range vars {
				if (v.ID == 0) == fresh {
					if err = tx.Create(&v).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

func validateStoryRules(vars []models.Variable, ports []models.Port, projectID uint) error {
	ids := map[uint]bool{}
	names := map[string]bool{}
	for _, v := range vars {
		if v.ProjectID != projectID || v.ID == 0 || ids[v.ID] || strings.TrimSpace(v.Name) == "" || names[v.Name] || !validStoryValue(v.InitialValue) {
			return errors.New("变量定义无效")
		}
		ids[v.ID] = true
		names[v.Name] = true
	}
	for _, p := range ports {
		if err := validateRules(vars, p.ConditionMode, p.Conditions, p.Effects); err != nil {
			return fmt.Errorf("端口「%s」：%w", p.Name, err)
		}
	}
	return nil
}
