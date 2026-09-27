package viewer

import (
	"strings"

	"github.com/henrylau/zip-photo-viewer/internal/loader"
)

type treeNode struct {
	name     string
	path     string
	index    int // -1 for folders
	children []*treeNode
}

func splitAlbumPath(p string) []string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.Trim(p, "/")
	if p == "" || p == "." {
		return nil
	}
	parts := strings.Split(p, "/")
	out := parts[:0]
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		out = append(out, part)
	}
	return out
}

func commonDirPrefix(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	if len(paths) == 1 {
		parts := splitAlbumPath(paths[0])
		if len(parts) <= 1 {
			return nil
		}
		return parts[:len(parts)-1]
	}

	prefix := splitAlbumPath(paths[0])
	for _, p := range paths[1:] {
		other := splitAlbumPath(p)
		n := min(len(prefix), len(other))
		i := 0
		for i < n && prefix[i] == other[i] {
			i++
		}
		prefix = prefix[:i]
		if len(prefix) == 0 {
			break
		}
	}
	return prefix
}

func insertAlbumNode(parent *treeNode, parts []string, index int) {
	if len(parts) == 0 {
		return
	}
	if len(parts) == 1 {
		parent.children = append(parent.children, &treeNode{
			name:  parts[0],
			path:  joinAlbumChild(parent.path, parts[0]),
			index: index,
		})
		return
	}

	var folder *treeNode
	for _, c := range parent.children {
		if c.index < 0 && c.name == parts[0] {
			folder = c
			break
		}
	}
	if folder == nil {
		folder = &treeNode{
			name:  parts[0],
			path:  joinAlbumChild(parent.path, parts[0]),
			index: -1,
		}
		parent.children = append(parent.children, folder)
	}
	insertAlbumNode(folder, parts[1:], index)
}

func joinAlbumChild(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "/" + name
}

func buildAlbumTree(entries []loader.FileInfo) []*treeNode {
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.FilePath
	}
	prefix := commonDirPrefix(paths)

	root := &treeNode{index: -1}
	for i, e := range entries {
		parts := splitAlbumPath(e.FilePath)
		if len(prefix) > 0 && len(parts) >= len(prefix) {
			same := true
			for j, p := range prefix {
				if parts[j] != p {
					same = false
					break
				}
			}
			if same {
				parts = parts[len(prefix):]
			}
		}
		if len(parts) == 0 {
			parts = []string{e.Name}
		}
		insertAlbumNode(root, parts, i)
	}
	return root.children
}

func findNodePath(nodes []*treeNode, index int) []string {
	for _, n := range nodes {
		if n.index == index {
			return []string{n.path}
		}
		if n.index < 0 {
			if sub := findNodePath(n.children, index); sub != nil {
				return append([]string{n.path}, sub...)
			}
		}
	}
	return nil
}

type treeRow struct {
	node  *treeNode
	depth int
}

func flattenAlbumTree(nodes []*treeNode, expanded map[string]bool) []treeRow {
	var rows []treeRow
	var walk func([]*treeNode, int)
	walk = func(nodes []*treeNode, depth int) {
		for _, n := range nodes {
			rows = append(rows, treeRow{node: n, depth: depth})
			if n.index < 0 && expanded[n.path] {
				walk(n.children, depth+1)
			}
		}
	}
	walk(nodes, 0)
	return rows
}
