package pages

import (
	"sort"

	"github.com/Microindole/quell/internal/core"
)

func BuildTree(procs []core.Process) []core.Process {
	childrenMap := make(map[int32][]*core.Process)
	var nodes []*core.Process
	for i := range procs {
		nodes = append(nodes, &procs[i])
	}

	var roots []*core.Process
	exists := make(map[int32]bool)
	for _, p := range nodes {
		exists[p.PID] = true
	}

	for _, p := range nodes {
		if p.PPID == 0 || !exists[p.PPID] || p.PID == p.PPID {
			roots = append(roots, p)
		} else {
			childrenMap[p.PPID] = append(childrenMap[p.PPID], p)
		}
	}

	sortFunc := func(list []*core.Process) {
		sort.Slice(list, func(i, j int) bool { return list[i].PID < list[j].PID })
	}
	sortFunc(roots)
	for _, children := range childrenMap {
		sortFunc(children)
	}

	var result []core.Process

	var traverse func(nodes []*core.Process, prefix string)
	traverse = func(nodes []*core.Process, prefix string) {
		for i, node := range nodes {
			isLast := i == len(nodes)-1

			// 👇 修改 1：使用更紧凑的连接符 (2个字符宽度)
			connector := "├─"
			if isLast {
				connector = "└─"
			}

			// 👇 修改 2：缩进也改为 2 个字符宽度，与上面对齐
			childPrefix := prefix + "│ "
			if isLast {
				childPrefix = prefix + "  "
			}

			newItem := *node
			newItem.TreePrefix = prefix + connector
			result = append(result, newItem)

			if children, ok := childrenMap[node.PID]; ok {
				traverse(children, childPrefix)
			}
		}
	}

	for _, root := range roots {
		newItem := *root
		newItem.TreePrefix = ""
		result = append(result, newItem)
		if children, ok := childrenMap[root.PID]; ok {
			traverse(children, "")
		}
	}

	return result
}
