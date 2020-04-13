// Copyright 2019 Keybase Inc. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package libfs

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/keybase/client/go/kbfs/data"
	"github.com/keybase/client/go/kbfs/kbfsblock"
	"github.com/keybase/client/go/kbfs/kbfsmd"
	"github.com/keybase/client/go/kbfs/libkbfs"
	"github.com/keybase/client/go/logger"
	"github.com/keybase/client/go/protocol/keybase1"
	"github.com/pkg/errors"
	billy "gopkg.in/src-d/go-billy.v4"
)

type statusFileNode struct {
	libkbfs.Node

	fb     data.FolderBranch
	config libkbfs.Config
	log    logger.Logger
}

var _ libkbfs.Node = (*statusFileNode)(nil)

func (sfn *statusFileNode) GetFile(ctx context.Context) billy.File {
	return &wrappedReadFile{
		name: StatusFileName,
		reader: func(ctx context.Context) ([]byte, time.Time, error) {
			return GetEncodedFolderStatus(ctx, sfn.config, sfn.fb)
		},
		log: sfn.log,
	}
}

func (sfn *statusFileNode) FillCacheDuration(d *time.Duration) {
	// Suggest kindly that no one should cache this node, since it
	// could change each time it's read.
	*d = 0
}

var updateHistoryRevsRE = regexp.MustCompile("^\\.([0-9]+)(-([0-9]+))?$") //nolint (`\.` doesn't seem to work in single quotes)

type updateHistoryFileNode struct {
	libkbfs.Node

	fb     data.FolderBranch
	config libkbfs.Config
	log    logger.Logger
	start  kbfsmd.Revision
	end    kbfsmd.Revision
}

var _ libkbfs.Node = (*updateHistoryFileNode)(nil)

func (uhfn updateHistoryFileNode) GetFile(ctx context.Context) billy.File {
	return &wrappedReadFile{
		name: StatusFileName,
		reader: func(ctx context.Context) ([]byte, time.Time, error) {
			return GetEncodedUpdateHistory(
				ctx, uhfn.config, uhfn.fb, uhfn.start, uhfn.end)
		},
		log: uhfn.log,
	}
}

func (uhfn *updateHistoryFileNode) FillCacheDuration(d *time.Duration) {
	// Suggest kindly that no one should cache this node, since it
	// could change each time it's read.
	*d = 0
}

type profileNode struct {
	libkbfs.Node

	config libkbfs.Config
	name   string
}

var _ libkbfs.Node = (*profileNode)(nil)

func (pn *profileNode) GetFile(ctx context.Context) billy.File {
	fs := NewProfileFS(pn.config)
	f, err := fs.Open(pn.name)
	if err != nil {
		return nil
	}
	return f
}

func (pn *profileNode) FillCacheDuration(d *time.Duration) {
	// Suggest kindly that no one should cache this node, since it
	// could change each time it's read.
	*d = 0
}

type profileListNode struct {
	libkbfs.Node

	config libkbfs.Config
}

var _ libkbfs.Node = (*profileListNode)(nil)

func (pln *profileListNode) ShouldCreateMissedLookup(
	ctx context.Context, name data.PathPartString) (
	bool, context.Context, data.EntryType, os.FileInfo, data.PathPartString,
	data.BlockPointer) {
	namePlain := name.Plaintext()

	fs := NewProfileFS(pln.config)
	fi, err := fs.Lstat(namePlain)
	if err != nil {
		return pln.Node.ShouldCreateMissedLookup(ctx, name)
	}

	return true, ctx, data.FakeFile, fi, data.PathPartString{}, data.ZeroPtr
}

func (pln *profileListNode) WrapChild(child libkbfs.Node) libkbfs.Node {
	child = pln.Node.WrapChild(child)
	return &profileNode{child, pln.config, child.GetBasename().Plaintext()}
}

func (pln *profileListNode) GetFS(ctx context.Context) libkbfs.NodeFSReadOnly {
	return NewProfileFS(pln.config)
}

// specialFileNode is a Node wrapper around a TLF node, that causes
// special files to be fake-created when they are accessed.
type specialFileNode struct {
	libkbfs.Node

	config libkbfs.Config
	log    logger.Logger
}

var _ libkbfs.Node = (*specialFileNode)(nil)

var perTlfWrappedNodeNames = map[string]bool{
	StatusFileName:        true,
	UpdateHistoryFileName: true,
	ProfileListDirName:    true,
}

var perTlfWrappedNodePrefixes = []string{
	UpdateHistoryFileName,
	DirBlockPrefix,
}

func shouldBeTlfWrappedNode(name string) bool {
	for _, p := range perTlfWrappedNodePrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return perTlfWrappedNodeNames[name]
}

func (sfn *specialFileNode) newUpdateHistoryFileNode(
	node libkbfs.Node, name string) *updateHistoryFileNode {
	revs := strings.TrimPrefix(name, UpdateHistoryFileName)
	if revs == "" {
		return &updateHistoryFileNode{
			Node:   node,
			fb:     sfn.GetFolderBranch(),
			config: sfn.config,
			log:    sfn.log,
			start:  kbfsmd.RevisionInitial,
			end:    kbfsmd.RevisionUninitialized,
		}
	}

	matches := updateHistoryRevsRE.FindStringSubmatch(revs)
	if len(matches) != 4 {
		return nil
	}

	start, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return nil
	}
	end := start
	if matches[3] != "" {
		end, err = strconv.ParseUint(matches[3], 10, 64)
		if err != nil {
			return nil
		}
	}

	return &updateHistoryFileNode{
		Node:   node,
		fb:     sfn.GetFolderBranch(),
		config: sfn.config,
		log:    sfn.log,
		start:  kbfsmd.Revision(start),
		end:    kbfsmd.Revision(end),
	}
}

// parseBlockPointer returns a real BlockPointer given a string.  The
// format for the string is: id.keyGen.dataVer.creatorUID.directType
func parseBlockPointer(plain string) (data.BlockPointer, error) {
	s := strings.Split(plain, ".")
	if len(s) != 5 {
		return data.ZeroPtr, errors.Errorf(
			"%s is not in the right format for a block pointer", plain)
	}

	id, err := kbfsblock.IDFromString(s[0])
	if err != nil {
		return data.ZeroPtr, err
	}

	keyGen, err := strconv.Atoi(s[1])
	if err != nil {
		return data.ZeroPtr, err
	}

	dataVer, err := strconv.Atoi(s[2])
	if err != nil {
		return data.ZeroPtr, err
	}

	creator, err := keybase1.UserOrTeamIDFromString(s[3])
	if err != nil {
		return data.ZeroPtr, err
	}

	directType := data.BlockDirectTypeFromString(s[4])

	return data.BlockPointer{
		ID:         id,
		KeyGen:     kbfsmd.KeyGen(keyGen),
		DataVer:    data.Ver(dataVer),
		DirectType: directType,
		Context: kbfsblock.MakeFirstContext(
			creator, keybase1.BlockType_DATA),
	}, nil
}

// ShouldCreateMissedLookup implements the Node interface for
// specialFileNode.
func (sfn *specialFileNode) ShouldCreateMissedLookup(
	ctx context.Context, name data.PathPartString) (
	bool, context.Context, data.EntryType, os.FileInfo, data.PathPartString,
	data.BlockPointer) {
	plain := name.Plaintext()
	if !shouldBeTlfWrappedNode(plain) {
		return sfn.Node.ShouldCreateMissedLookup(ctx, name)
	}

	switch {
	case plain == StatusFileName:
		sfn := &statusFileNode{
			Node:   nil,
			fb:     sfn.GetFolderBranch(),
			config: sfn.config,
			log:    sfn.log,
		}
		f := sfn.GetFile(ctx)
		return true, ctx, data.FakeFile, f.(*wrappedReadFile).GetInfo(),
			data.PathPartString{}, data.ZeroPtr
	case plain == ProfileListDirName:
		return true, ctx, data.FakeDir,
			&wrappedReadFileInfo{plain, 0, sfn.config.Clock().Now(), true},
			data.PathPartString{}, data.ZeroPtr
	case strings.HasPrefix(plain, UpdateHistoryFileName):
		uhfn := sfn.newUpdateHistoryFileNode(nil, plain)
		if uhfn == nil {
			return sfn.Node.ShouldCreateMissedLookup(ctx, name)
		}
		f := uhfn.GetFile(ctx)
		return true, ctx, data.FakeFile, f.(*wrappedReadFile).GetInfo(),
			data.PathPartString{}, data.ZeroPtr
	case strings.HasPrefix(plain, DirBlockPrefix):
		ptr, err := parseBlockPointer(strings.TrimPrefix(plain, DirBlockPrefix))
		if err != nil {
			sfn.log.CDebugf(
				ctx, "Couldn't parse block pointer for %s: %+v", name, err)
			return sfn.Node.ShouldCreateMissedLookup(ctx, name)
		}

		info := &wrappedReadFileInfo{
			name:  plain,
			size:  0,
			mtime: time.Now(),
			dir:   true,
		}

		return true, ctx, data.RealDir, info, data.PathPartString{}, ptr
	default:
		panic(fmt.Sprintf("Name %s was in map, but not in switch", name))
	}

}

// WrapChild implements the Node interface for specialFileNode.
func (sfn *specialFileNode) WrapChild(child libkbfs.Node) libkbfs.Node {
	child = sfn.Node.WrapChild(child)
	name := child.GetBasename().Plaintext()
	if !shouldBeTlfWrappedNode(name) {
		if child.EntryType() == data.Dir {
			// Wrap this child too, so we can look up special files in
			// subdirectories of this node as well.
			return &specialFileNode{
				Node:   child,
				config: sfn.config,
				log:    sfn.log,
			}
		}
		return child
	}

	switch {
	case name == StatusFileName:
		return &statusFileNode{
			Node:   &libkbfs.ReadonlyNode{Node: child},
			fb:     sfn.GetFolderBranch(),
			config: sfn.config,
			log:    sfn.log,
		}
	case name == ProfileListDirName:
		return &profileListNode{
			Node:   &libkbfs.ReadonlyNode{Node: child},
			config: sfn.config,
		}
	case strings.HasPrefix(name, UpdateHistoryFileName):
		uhfn := sfn.newUpdateHistoryFileNode(child, name)
		if uhfn == nil {
			return child
		}
		return uhfn
	case strings.HasPrefix(name, DirBlockPrefix):
		return &libkbfs.ReadonlyNode{Node: child}
	default:
		panic(fmt.Sprintf("Name %s was in map, but not in switch", name))
	}
}

// rootWrapper is a struct that manages wrapping root nodes with
// special per-TLF content.
type rootWrapper struct {
	config libkbfs.Config
	log    logger.Logger
}

func (rw rootWrapper) wrap(node libkbfs.Node) libkbfs.Node {
	return &specialFileNode{
		Node:   node,
		config: rw.config,
		log:    rw.log,
	}
}

// AddRootWrapper should be called on startup by any KBFS interface
// that wants to handle special files.
func AddRootWrapper(config libkbfs.Config) {
	rw := rootWrapper{config, config.MakeLogger("")}
	config.AddRootNodeWrapper(rw.wrap)
}
