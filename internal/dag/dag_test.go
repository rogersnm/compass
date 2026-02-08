package dag

import (
	"testing"

	"github.com/rogersnm/compass/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func task(id string, deps ...string) *model.Task {
	return &model.Task{ID: id, Status: model.StatusOpen, DependsOn: deps}
}

func TestBuildFromTasks_Empty(t *testing.T) {
	g := BuildFromTasks(nil)
	assert.Empty(t, g.Roots())
}

func TestBuildFromTasks_SingleTask(t *testing.T) {
	g := BuildFromTasks([]*model.Task{task("A")})
	assert.Equal(t, []string{"A"}, g.Roots())
	assert.Equal(t, []string{"A"}, g.Leaves())
}

func TestBuildFromTasks_LinearChain(t *testing.T) {
	// C depends on B, B depends on A
	g := BuildFromTasks([]*model.Task{
		task("A"),
		task("B", "A"),
		task("C", "B"),
	})
	assert.Equal(t, []string{"A"}, g.Roots())
	assert.Equal(t, []string{"C"}, g.Leaves())
}

func TestValidateAcyclic_ValidDAG(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A"),
		task("B", "A"),
		task("C", "A"),
		task("D", "B", "C"),
	})
	assert.NoError(t, g.ValidateAcyclic())
}

func TestValidateAcyclic_SimpleCycle(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A", "B"),
		task("B", "A"),
	})
	err := g.ValidateAcyclic()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestValidateAcyclic_LongCycle(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A", "D"),
		task("B", "A"),
		task("C", "B"),
		task("D", "C"),
	})
	assert.Error(t, g.ValidateAcyclic())
}

func TestValidateAcyclic_SelfLoop(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A", "A"),
	})
	assert.Error(t, g.ValidateAcyclic())
}

func TestValidateAcyclic_DiamondNoCycle(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A"),
		task("B", "A"),
		task("C", "A"),
		task("D", "B", "C"),
	})
	assert.NoError(t, g.ValidateAcyclic())
}

func TestTopologicalSort_LinearChain(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A"),
		task("B", "A"),
		task("C", "B"),
	})
	result, err := g.TopologicalSort()
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C"}, result)
}

func TestTopologicalSort_Diamond(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A"),
		task("B", "A"),
		task("C", "A"),
		task("D", "B", "C"),
	})
	result, err := g.TopologicalSort()
	require.NoError(t, err)
	// A must come first, D must come last
	assert.Equal(t, "A", result[0])
	assert.Equal(t, "D", result[3])
}

func TestTopologicalSort_Disconnected(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A"),
		task("B"),
		task("C"),
	})
	result, err := g.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestTransitiveDeps(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A"),
		task("B", "A"),
		task("C", "B"),
	})
	deps := g.TransitiveDeps("C")
	assert.Contains(t, deps, "B")
	assert.Contains(t, deps, "A")
}

func TestDependents(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A"),
		task("B", "A"),
		task("C", "A"),
	})
	deps := g.Dependents("A")
	assert.ElementsMatch(t, []string{"B", "C"}, deps)
}

func TestRoots(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A"),
		task("B", "A"),
		task("C"),
	})
	assert.ElementsMatch(t, []string{"A", "C"}, g.Roots())
}

func TestLeaves(t *testing.T) {
	g := BuildFromTasks([]*model.Task{
		task("A"),
		task("B", "A"),
	})
	assert.Equal(t, []string{"B"}, g.Leaves())
}
