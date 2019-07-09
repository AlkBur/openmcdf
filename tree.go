package openmcdf

import "strings"

type Tree struct {
	root *Node
	size int
}

type Node struct {
	left, right, parent *Node
	color               int
	Key                 string
	Value               *Directory
	modified            bool
}

func NewTree(root *Node) *Tree {
	return &Tree{root: root}
}

func (this *Tree) Size() int {
	return this.size
}

func (this *Tree) GetRoot() *Node {
	return this.root
}

func NewNode(de *Directory) *Node {
	return &Node{Value: de, Key: de.Name()}
}

func (this *Tree) Insert(z *Node) {
	x := this.root
	var y *Node

	for x != nil {
		y = x
		//LessThan
		if strings.Compare(z.Key, x.Key) > 0 {
			x = x.left
		} else {
			x = x.right
		}
	}

	z.parent = y
	z.color = Red
	z.modified = true

	this.size++

	if y == nil {
		z.color = Black
		this.root = z
		return
		//LessThan
	} else if strings.Compare(z.Key, y.Key) > 0 {
		y.left = z
		y.modified = true
	} else {
		y.right = z
		y.modified = true
	}
	this.rbInsertFixup(z)
}

// Delete deletes the node by key
func (t *Tree) Delete(z *Node) {
	if z == nil {
		return
	}

	var x, y, parent *Node
	y = z
	yOriginalColor := y.color
	parent = z.parent
	if z.left == nil {
		x = z.right
		t.transplant(z, z.right)
	} else if z.right == nil {
		x = z.left
		t.transplant(z, z.left)
	} else {
		y = minimum(z.right)
		yOriginalColor = y.color
		x = y.right

		if y.parent == z {
			if x == nil {
				parent = y
			} else {
				x.parent = y
			}
		} else {
			t.transplant(y, y.right)
			y.right = z.right
			y.right.parent = y
			y.modified = true
		}
		t.transplant(z, y)
		y.left = z.left
		y.left.parent = y
		y.color = z.color
		y.modified = true
	}

	if yOriginalColor == Black {
		t.rbDeleteFixup(x, parent)
	}
	t.size--
}

func (this *Tree) rbInsertFixup(z *Node) {
	var y *Node
	for z.parent != nil && z.parent.color == Red {
		if z.parent == z.parent.parent.left {
			y = z.parent.parent.right
			if y != nil && y.color == Red {
				z.parent.color = Black
				z.parent.modified = true
				y.color = Black
				y.modified = true
				z.parent.parent.color = Red
				z.parent.parent.modified = true
				z = z.parent.parent
			} else {
				if z == z.parent.right {
					z = z.parent
					this.leftRotate(z)
				}
				z.parent.color = Black
				z.parent.modified = true
				z.parent.parent.color = Red
				z.parent.parent.modified = true
				this.rightRotate(z.parent.parent)
			}
		} else {
			y = z.parent.parent.left
			if y != nil && y.color == Red {
				z.parent.color = Black
				z.parent.modified = true
				y.color = Black
				y.modified = true
				z.parent.parent.color = Red
				z.parent.parent.modified = true
				z = z.parent.parent
			} else {
				if z == z.parent.left {
					z = z.parent
					this.rightRotate(z)
				}
				z.parent.color = Black
				z.parent.modified = true
				z.parent.parent.color = Red
				z.parent.parent.modified = true
				this.leftRotate(z.parent.parent)
			}
		}
	}
	this.root.color = Black
	this.root.modified = true
}

func (t *Tree) rbDeleteFixup(x, parent *Node) {
	var w *Node

	for x != t.root && getColor(x) == Black {
		if x != nil {
			parent = x.parent
		}
		if x == parent.left {
			w = parent.right
			if w.color == Red {
				w.color = Black
				w.modified = true
				parent.color = Red
				parent.modified = true
				t.leftRotate(parent)
				w = parent.right
			}
			if getColor(w.left) == Black && getColor(w.right) == Black {
				w.color = Red
				w.modified = true
				x = parent
			} else {
				if getColor(w.right) == Black {
					if w.left != nil {
						w.left.color = Black
						w.left.modified = true
					}
					w.color = Red
					w.modified = true
					t.rightRotate(w)
					w = parent.right
				}
				w.color = parent.color
				parent.color = Black
				parent.modified = true
				if w.right != nil {
					w.right.color = Black
					w.right.modified = true
				}
				t.leftRotate(parent)
				x = t.root
			}
		} else {
			w = parent.left
			if w.color == Red {
				w.color = Black
				w.modified = true
				parent.color = Red
				parent.modified = true
				t.rightRotate(parent)
				w = parent.left
			}
			if getColor(w.left) == Black && getColor(w.right) == Black {
				w.color = Red
				w.modified = true
				x = parent
			} else {
				if getColor(w.left) == Black {
					if w.right != nil {
						w.right.color = Black
					}
					w.color = Red
					w.modified = true
					t.leftRotate(w)
					w = parent.left
				}
				w.color = parent.color
				w.modified = true
				parent.color = Black
				parent.modified = true
				if w.left != nil {
					w.left.color = Black
					w.left.modified = true
				}
				t.rightRotate(parent)
				x = t.root
			}
		}
	}
	if x != nil {
		x.color = Black
		x.modified = true
	}
}

func (this *Tree) leftRotate(x *Node) {
	y := x.right
	x.right = y.left
	x.modified = true
	if y.left != nil {
		y.left.parent = x
	}
	y.parent = x.parent
	if x.parent == nil {
		this.root = y
	} else if x == x.parent.left {
		x.parent.left = y
		x.parent.modified = true
	} else {
		x.parent.right = y
		x.parent.modified = true
	}
	y.left = x
	y.modified = true
	x.parent = y
}

func (this *Tree) rightRotate(x *Node) {
	y := x.left
	x.left = y.right
	x.modified = true
	if y.right != nil {
		y.right.parent = x
	}
	y.parent = x.parent
	if x.parent == nil {
		this.root = y
	} else if x == x.parent.left {
		x.parent.left = y
		x.parent.modified = true
	} else {
		x.parent.right = y
		x.parent.modified = true
	}
	y.right = x
	y.modified = true
	x.parent = y
}

func (t *Tree) Find(key string) *Directory {
	n := t.findnode(key)
	if n != nil {
		return n.Value
	}
	return nil
}

func (t *Tree) findnode(key string) *Node {
	x := t.root
	for x != nil {
		//LessThan
		if strings.Compare(key, x.Key) > 0 {
			x = x.left
		} else {
			if key == x.Key {
				return x
			}
			x = x.right
		}
	}
	return nil
}

// transplant transplants the subtree u and v
func (t *Tree) transplant(u, v *Node) {
	if u.parent == nil {
		t.root = v
	} else if u == u.parent.left {
		u.parent.left = v
		u.parent.modified = true
	} else {
		u.parent.right = v
		u.parent.modified = true
	}
	if v == nil {
		return
	}
	v.parent = u.parent
}

func (t *Tree) Iterator() *Node {
	return minimum(t.root)
}

func (n *Node) Next() *Node {
	return successor(n)
}

func minimum(n *Node) *Node {
	for n.left != nil {
		n = n.left
	}
	return n
}

func successor(x *Node) *Node {
	if x.right != nil {
		return minimum(x.right)
	}
	y := x.parent
	for y != nil && x == y.right {
		x = y
		y = x.parent
	}
	return y
}

func (n *Node) modifiedValue() bool {
	n.modified = false
	m := false
	if uint8(n.color) != n.Value.colorFlag {
		n.Value.colorFlag = uint8(n.color)
		m = true
	}
	left := NOSTREAM
	if n.left != nil {
		left = uint32(n.left.Value.id)
	}
	if n.Value.leftSiblingID != left {
		n.Value.leftSiblingID = left
		m = true
	}
	right := NOSTREAM
	if n.right != nil {
		right = uint32(n.right.Value.id)
	}
	if n.Value.rightSiblingID != right {
		n.Value.rightSiblingID = right
		m = true
	}
	return m
}

func (n *Node) GetColor() int {
	if n == nil {
		return Black
	}
	return n.color
}

func (this *Node) GetLeft() *Node {
	return this.left
}

func (this *Node) GetRight() *Node {
	return this.right
}

func (this *Node) GetParent() *Node {
	return this.parent
}

func getColor(n *Node) int {
	return n.GetColor()
}
