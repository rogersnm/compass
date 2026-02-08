package dag

import (
	"fmt"
	"sort"

	"github.com/rogersnm/compass/internal/model"
)

type Graph struct {
	nodes map[string]*model.Task
	edges map[string][]string // task -> depends_on
	rev   map[string][]string // task -> dependents
}

func BuildFromTasks(tasks []*model.Task) *Graph {
	g := &Graph{
		nodes: make(map[string]*model.Task),
		edges: make(map[string][]string),
		rev:   make(map[string][]string),
	}
	for _, t := range tasks {
		g.nodes[t.ID] = t
		g.edges[t.ID] = append([]string{}, t.DependsOn...)
		for _, dep := range t.DependsOn {
			g.rev[dep] = append(g.rev[dep], t.ID)
		}
	}
	return g
}

// ValidateAcyclic checks for cycles using DFS. Returns an error describing
// the cycle path if one exists.
func (g *Graph) ValidateAcyclic() error {
	const (
		white = 0 // unvisited
		gray  = 1 // in current path
		black = 2 // finished
	)

	color := make(map[string]int)
	parent := make(map[string]string)

	var dfs func(node string) error
	dfs = func(node string) error {
		color[node] = gray
		for _, dep := range g.edges[node] {
			if _, ok := g.nodes[dep]; !ok {
				continue
			}
			if color[dep] == gray {
				cycle := buildCyclePath(parent, node, dep)
				return fmt.Errorf("cycle detected: %s", cycle)
			}
			if color[dep] == white {
				parent[dep] = node
				if err := dfs(dep); err != nil {
					return err
				}
			}
		}
		color[node] = black
		return nil
	}

	for id := range g.nodes {
		if color[id] == white {
			if err := dfs(id); err != nil {
				return err
			}
		}
	}
	return nil
}

func buildCyclePath(parent map[string]string, from, to string) string {
	path := []string{to}
	cur := from
	for cur != to {
		path = append(path, cur)
		cur = parent[cur]
	}
	path = append(path, to)
	// Reverse to show in dependency order
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	result := path[0]
	for _, p := range path[1:] {
		result += " -> " + p
	}
	return result
}

// TopologicalSort returns tasks in dependency order using Kahn's algorithm.
func (g *Graph) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int)
	for id := range g.nodes {
		inDegree[id] = 0
	}
	for id := range g.nodes {
		for _, dep := range g.edges[id] {
			if _, ok := g.nodes[dep]; ok {
				inDegree[id]++
			}
		}
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		for _, dependent := range g.rev[node] {
			if _, ok := g.nodes[dependent]; !ok {
				continue
			}
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
				sort.Strings(queue)
			}
		}
	}

	if len(result) != len(g.nodes) {
		return nil, fmt.Errorf("cycle detected: topological sort incomplete")
	}
	return result, nil
}

func (g *Graph) TransitiveDeps(id string) []string {
	visited := make(map[string]bool)
	var result []string
	var walk func(string)
	walk = func(node string) {
		for _, dep := range g.edges[node] {
			if !visited[dep] {
				visited[dep] = true
				result = append(result, dep)
				walk(dep)
			}
		}
	}
	walk(id)
	return result
}

func (g *Graph) Dependents(id string) []string {
	return g.rev[id]
}

func (g *Graph) Roots() []string {
	var roots []string
	for id := range g.nodes {
		if len(g.edges[id]) == 0 {
			roots = append(roots, id)
		}
	}
	sort.Strings(roots)
	return roots
}

func (g *Graph) Leaves() []string {
	var leaves []string
	for id := range g.nodes {
		if len(g.rev[id]) == 0 {
			leaves = append(leaves, id)
		}
	}
	sort.Strings(leaves)
	return leaves
}

func (g *Graph) Node(id string) *model.Task {
	return g.nodes[id]
}
