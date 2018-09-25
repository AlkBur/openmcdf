package openmcdf

import (
	"errors"
	"fmt"
)

type Storage struct {
	cf   *CompoundFile
	de   *Directory
	tree *Tree
}

func newStorage(de *Directory, cf *CompoundFile) *Storage {
	this := &Storage{cf: cf, de: de}
	return this
}

func (this *Storage) GetStream(name string) (*Stream, error) {
	if this == nil || this.de == nil {
		return nil, fmt.Errorf("The storage directory is nil")
	}
	if this.tree == nil {
		this.loadChildren()
	}
	de := this.tree.Find(name)
	if de == nil {
		return nil, StreamNotFound
	}
	return de.newStream(this.cf), nil
}

func (this *Storage) find(de, find *Directory) *Directory {
	if de == nil || find == nil {
		return nil
	}
	cmp := de.compareTo(find)
	if cmp < 0 {
		de = this.cf.directory.getLeft(de)
		de = this.find(de, find)
	} else if cmp > 0 {
		de = this.cf.directory.getRight(de)
		de = this.find(de, find)
	}
	return de
}

func (this *Storage) String() string {
	return this.de.String()
}

func (this *Storage) AddStream(name string) (*Stream, error) {
	if this == nil {
		return nil, errors.New("Error add stream: storage is nil")
	}

	st := newStream(name, this.cf)
	de := st.de

	//Add
	this.cf.directory.Add(de)
	//tree
	if this.tree == nil {
		this.tree = NewTree(nil)
	}
	node := NewNode(de)
	this.tree.Insert(node)
	//update
	it := this.tree.Iterator()
	for it != nil {
		if it.modified {
			if it.modifiedValue() || it == node {
				if err := this.cf.updateDirectory(it.Value); err != nil {
					return nil, err
				}
			}
		}
		it = it.Next()
	}
	if uint32(this.tree.root.Value.id) != this.de.childID {
		this.de.childID = uint32(this.tree.root.Value.id)
		if err := this.cf.updateDirectory(this.de); err != nil {
			return nil, err
		}
	}
	return st, nil
}

func (this *Storage) loadChildren() {
	de := this.cf.directory.getChild(this.de)
	if de != nil {
		node := NewNode(de)
		this.tree = NewTree(node)
		this.loadSiblings(node)
	} else {
		this.tree = NewTree(nil)
	}
}

func (this *Storage) loadSiblings(node *Node) {
	var nodeLeft *Node
	var nodeRight *Node
	if node == nil {
		return
	}
	//get left and right directory
	deLeft := this.cf.directory.getLeft(node.Value)
	deRight := this.cf.directory.getRight(node.Value)
	if deLeft != nil {
		nodeLeft = NewNode(deLeft)
		nodeLeft.parent = node
	}
	if deRight != nil {
		nodeRight = NewNode(deRight)
		nodeRight.parent = node
	}
	node.left = nodeLeft
	node.right = nodeRight
	//------------------------
	this.loadSiblings(node.left)
	this.loadSiblings(node.right)
}
