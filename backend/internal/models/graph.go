package models

import "fmt"

// ValidateGraph validates the ordinary tree; magic edges never affect depth or ownership.
// Level 1 identifies the unique root; other levels are derived from ordinary connections.
func ValidateGraph(scenes []Node, ports []Port) error {
	byID, byHash := map[uint]*Node{}, map[string]*Node{}
	var root *Node
	for i := range scenes {
		s := &scenes[i]
		if s.Hash == "" || s.Name == "" || byID[s.ID] != nil || byHash[s.Hash] != nil {
			return fmt.Errorf("节点标识或名称无效")
		}
		byID[s.ID], byHash[s.Hash] = s, s
		if s.Level == 1 {
			if root != nil {
				return fmt.Errorf("每个工程必须恰好有一个根节点")
			}
			root = s
		}
	}
	if root == nil {
		return fmt.Errorf("工程缺少根节点")
	}
	children := map[uint][]uint{}
	incoming := map[uint]int{} // 普通入端口数量仅在校验期间统计，不写入节点记录。
	ids, hashes := map[uint]bool{}, map[string]bool{}
	for _, p := range ports {
		from, to := byID[p.NodeID], byHash[p.ExportHash]
		if from == nil || to == nil || from.CanvasID != to.CanvasID {
			return fmt.Errorf("端口「%s」目标不存在或跨工程", p.Name)
		}
		if p.ID == 0 || p.Hash == "" || ids[p.ID] || hashes[p.Hash] || p.Name == "" {
			return fmt.Errorf("端口标识或文案无效")
		}
		if p.TargetKind != "mp4" && p.TargetKind != "html" {
			return fmt.Errorf("端口内容类型无效")
		}
		ids[p.ID], hashes[p.Hash] = true, true
		if p.Magic {
			continue
		}
		incoming[to.ID]++
		if to.ID == root.ID || incoming[to.ID] > 1 {
			return fmt.Errorf("根节点不能有普通入端口，其他节点至多一个普通入端口")
		}
		children[from.ID] = append(children[from.ID], to.ID)
	}
	root.Level = 1
	queue := []uint{root.ID}
	seen := map[uint]bool{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			return fmt.Errorf("普通连接不能形成环")
		}
		seen[id] = true
		for _, child := range children[id] {
			byID[child].Level = byID[id].Level + 1
			queue = append(queue, child)
		}
	}
	if len(seen) != len(scenes) {
		return fmt.Errorf("普通连接存在环或未连接分支，请保持节点连接到根节点")
	}
	return nil
}
