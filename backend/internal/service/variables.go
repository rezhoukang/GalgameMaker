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

// PortRules 作为一个整体保存，避免只保存了部分条件或效果。
type PortRules struct {
	ConditionMode string                 `json:"conditionMode"` // all / any
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
			return errors.New("效果数值必须为安全整数")
		}
		if e.Set != 0 && e.Set != 1 {
			return errors.New("set 必须为 0（增减）或 1（设值）")
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
	p.ConditionMode = rules.ConditionMode
	p.Conditions = append([]models.PortCondition{}, rules.Conditions...)
	p.Effects = append([]models.PortEffect{}, rules.Effects...)
	return nil
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
		// 条件和效果均从端口 JSON 读取，统一检查变量引用。
		var ports []models.Port
		if err = tx.Where("node_id IN (?)", tx.Model(&models.Node{}).Select("id").Where("project_id = ?", projectID)).Find(&ports).Error; err != nil {
			return err
		}
		used := map[uint]bool{}
		for _, p := range ports {
			for _, c := range p.Conditions {
				used[c.VariableID] = true
			}
			for _, e := range p.Effects {
				used[e.VariableID] = true
			}
		}
		for _, v := range current {
			if !kept[v.ID] && used[v.ID] {
				return fmt.Errorf("变量「%s」仍被端口条件或效果使用", v.Name)
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
