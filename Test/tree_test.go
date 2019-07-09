package Test

import (
	mcdf "github.com/AlkBur/openmcdf"
	"github.com/stretchr/testify/assert"
	"testing"
)

func GetDirectoryRepository(t *testing.T, count int) []*mcdf.Directory {
	repo := make([]*mcdf.Directory, 0, count)
	for i := 0; i < count; i++ {
		de := mcdf.NewDirectory()
		err := de.SetName(String(int32(i)))
		assert.NoError(t, err)

		repo = append(repo, de)
	}

	return repo
}

func Test_RBTREE_INSERT(t *testing.T) {
	rbTree := mcdf.NewTree(nil)
	repo := GetDirectoryRepository(t, 1000000)

	for _, item := range repo {
		node := mcdf.NewNode(item)
		rbTree.Insert(node)
	}

	for i := 0; i < len(repo); i++ {
		name := String(int32(i))
		c := rbTree.Find(name)
		assert.NotNil(t, c)
		assert.True(t, name == c.Name())
	}
}

func Test_RBTREE_DELETE(t *testing.T) {
	rbTree := mcdf.NewTree(nil)
	repo := GetDirectoryRepository(t, 25)

	for _, item := range repo {
		node := mcdf.NewNode(item)
		rbTree.Insert(node)
	}
	assert.True(t, rbTree.Size() == 25, "должно быть 25 - %v", rbTree.Size())

	{
		de := mcdf.NewDirectory()
		err := de.SetName("5")
		assert.NoError(t, err)

		node := mcdf.NewNode(de)
		rbTree.Delete(node)
	}
	{
		de := mcdf.NewDirectory()
		err := de.SetName("24")
		assert.NoError(t, err)

		node := mcdf.NewNode(de)
		rbTree.Delete(node)
	}
	{
		de := mcdf.NewDirectory()
		err := de.SetName("7")
		assert.NoError(t, err)

		node := mcdf.NewNode(de)
		rbTree.Delete(node)
	}

	assert.True(t, rbTree.Size() == 22, "Должно быть 22 - %v", rbTree.Size())
}

func VerifyProperties(tree *mcdf.Tree, t *testing.T) {
	VerifyProperty1(tree.GetRoot(), t)
	VerifyProperty2(tree.GetRoot(), t)
	VerifyProperty3(tree.GetRoot(), t)
	VerifyProperty4(tree.GetRoot(), t)
}

func VerifyProperty1(n *mcdf.Node, t *testing.T) {

	assert.True(t, n.GetColor() == mcdf.Red || n.GetColor() == mcdf.Black)

	if n == nil {
		return
	}

	VerifyProperty1(n.GetLeft(), t)
	VerifyProperty1(n.GetRight(), t)
}

func VerifyProperty2(root *mcdf.Node, t *testing.T) {
	assert.True(t, root.GetColor() == mcdf.Black)
}

func VerifyProperty3(n *mcdf.Node, t *testing.T) {

	if n.GetColor() == mcdf.Red {
		assert.True(t, n.GetLeft().GetColor() == mcdf.Black)
		assert.True(t, n.GetRight().GetColor() == mcdf.Black)
		assert.True(t, n.GetParent().GetColor() == mcdf.Black)
	}

	if n == nil {
		return
	}
	VerifyProperty4(n.GetLeft(), t)
	VerifyProperty4(n.GetRight(), t)
}

func VerifyProperty4Helper(t *testing.T, n *mcdf.Node, blackCount, pathBlackCount int) int {
	if n.GetColor() == mcdf.Black {
		blackCount++
	}
	if n == nil {
		if pathBlackCount == -1 {
			pathBlackCount = blackCount
		} else {
			assert.True(t, blackCount == pathBlackCount)

		}
		return pathBlackCount
	}

	pathBlackCount = VerifyProperty4Helper(t, n.GetLeft(), blackCount, pathBlackCount)
	pathBlackCount = VerifyProperty4Helper(t, n.GetRight(), blackCount, pathBlackCount)
	return pathBlackCount
}

func VerifyProperty4(root *mcdf.Node, t *testing.T) {
	VerifyProperty4Helper(t, root, 0, -1)
}

func Test_RBTREE_ENUMERATE(t *testing.T) {
	rbTree := mcdf.NewTree(nil)
	repo := GetDirectoryRepository(t, 10000)

	for _, item := range repo {
		node := mcdf.NewNode(item)
		rbTree.Insert(node)
	}

	VerifyProperties(rbTree, t)
}
