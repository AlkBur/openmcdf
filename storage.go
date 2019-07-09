package openmcdf

import (
	"errors"
	"fmt"
)

var NotFoundDirectory = errors.New("Directory or stream not found")

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
	de, err := this.getDirectory(name)
	if err != nil || de == nil {
		return nil, StreamNotFound
	} else if de.objectType != StgStream {
		return nil, fmt.Errorf("This directory isn't stream: %v", de)
	}
	return de.newStream(this.cf), nil
}

func (this *Storage) GetStorage(name string) (*Storage, error) {
	de, err := this.getDirectory(name)
	if err != nil || de == nil {
		return nil, NotFoundDirectory
	} else if de.objectType != StgStorage {
		return nil, fmt.Errorf("This directory isn't storage: %v", de)
	}
	return de.newStorage(this.cf), nil
}

func (this *Storage) getDirectory(name string) (*Directory, error) {
	if this == nil || this.de == nil {
		return nil, fmt.Errorf("The storage directory is nil")
	}
	if this.tree == nil {
		this.loadChildren()
	}
	return this.tree.Find(name), nil
}

func (this *Storage) String() string {
	return this.de.String()
}

func (this *Storage) AddStream(name string) (*Stream, error) {
	var err error
	if this == nil {
		err = errors.New("Storage is nil")
		return nil, err
	}
	//tree
	if this.tree == nil {
		this.loadChildren()
	}
	de := this.tree.Find(name)
	if de != nil {
		err = fmt.Errorf("A directory with this name already exists: %v", de)
		return nil, err
	}
	if de, err = this.cf.directory.New(this.cf, name, StgStream); err != nil {
		return nil, err
	}
	node := NewNode(de)
	node.modified = true
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
			it.modified = false
		}
		it = it.Next()
	}
	if uint32(this.tree.root.Value.id) != this.de.childID {
		this.de.childID = uint32(this.tree.root.Value.id)
		if err := this.cf.updateDirectory(this.de); err != nil {
			return nil, err
		}
	}
	return de.newStream(this.cf), err
}

func (this *Storage) AddStorage(name string) (*Storage, error) {
	var err error
	if this == nil {
		err = errors.New("Storage is nil")
		return nil, err
	}
	//tree
	if this.tree == nil {
		this.loadChildren()
	}
	de := this.tree.Find(name)
	if de != nil {
		err = fmt.Errorf("A directory with this name already exists: %v", de)
		return nil, err
	}
	if de, err = this.cf.directory.New(this.cf, name, StgStorage); err != nil {
		return nil, err
	}
	node := NewNode(de)
	node.modified = true
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
			it.modified = false
		}
		it = it.Next()
	}
	if uint32(this.tree.root.Value.id) != this.de.childID {
		this.de.childID = uint32(this.tree.root.Value.id)
		if err := this.cf.updateDirectory(this.de); err != nil {
			return nil, err
		}
	}
	return de.newStorage(this.cf), err
}

func (this *Storage) Delete(name string) (err error) {
	if this == nil {
		err = errors.New("Storage is nil")
		return
	}
	if this.tree == nil {
		this.loadChildren()
	}
	node := this.tree.findnode(name)
	if node == nil {
		err = NotFoundDirectory
		return
	}
	node.modified = true
	this.tree.Delete(node)

	//Update directory
	it := this.tree.Iterator()
	for it != nil {
		if it.modified {
			if it.modifiedValue() && it != node {
				if err = this.cf.updateDirectory(it.Value); err != nil {
					return
				}
			}
			it.modified = false
		}
		it = it.Next()
	}
	if uint32(this.tree.root.Value.id) != this.de.childID {
		this.de.childID = uint32(this.tree.root.Value.id)
		if err = this.cf.updateDirectory(this.de); err != nil {
			return
		}
	}
	//Clear directory
	if err = this.cf.directory.Push(node.Value); err != nil {
		return
	}
	if err = this.cf.updateDirectory(node.Value); err != nil {
		return
	}
	return
}

func (this *Storage) loadChildren() {
	de := this.cf.directory.getChild(this.de)
	this.tree = NewTree(nil)
	this.addNode(de)
}

func (this *Storage) addNode(de *Directory) {
	if de == nil {
		return
	}

	node := NewNode(de)
	this.tree.Insert(node)

	deLeft := this.cf.directory.getLeft(node.Value)
	deRight := this.cf.directory.getRight(node.Value)

	this.addNode(deLeft)
	this.addNode(deRight)
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
