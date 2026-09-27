package viewer

import (
	"reflect"
	"testing"

	"github.com/henrylau/zip-photo-viewer/internal/loader"
)

func TestCommonDirPrefix(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  []string
	}{
		{
			name:  "disk folder album",
			paths: []string{"/Users/me/photos/a.jpg", "/Users/me/photos/b.png"},
			want:  []string{"Users", "me", "photos"},
		},
		{
			name:  "nested archive",
			paths: []string{"album/ch1/01.jpg", "album/ch2/01.jpg"},
			want:  []string{"album"},
		},
		{
			name:  "same nested folder",
			paths: []string{"album/ch1/01.jpg", "album/ch1/02.jpg"},
			want:  []string{"album", "ch1"},
		},
		{
			name:  "single file",
			paths: []string{"/Users/me/photos/a.jpg"},
			want:  []string{"Users", "me", "photos"},
		},
		{
			name:  "windows separators",
			paths: []string{`C:\photos\a.jpg`, `C:\photos\b.jpg`},
			want:  []string{"C:", "photos"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commonDirPrefix(tt.paths)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("commonDirPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildAlbumTree_FlatFolder(t *testing.T) {
	tree := buildAlbumTree([]loader.FileInfo{
		{Name: "a.jpg", FilePath: "/Users/me/photos/a.jpg"},
		{Name: "b.png", FilePath: "/Users/me/photos/b.png"},
	})
	if len(tree) != 2 {
		t.Fatalf("expected 2 root files, got %d", len(tree))
	}
	if tree[0].name != "a.jpg" || tree[0].index != 0 || tree[0].children != nil {
		t.Fatalf("first node = %+v", tree[0])
	}
	if tree[1].name != "b.png" || tree[1].index != 1 {
		t.Fatalf("second node = %+v", tree[1])
	}
}

func TestBuildAlbumTree_NestedArchive(t *testing.T) {
	tree := buildAlbumTree([]loader.FileInfo{
		{Name: "01.jpg", FilePath: "album/ch1/01.jpg"},
		{Name: "02.jpg", FilePath: "album/ch1/02.jpg"},
		{Name: "01.jpg", FilePath: "album/ch2/01.jpg"},
	})
	if len(tree) != 2 {
		t.Fatalf("expected 2 chapter folders, got %d", len(tree))
	}
	if tree[0].name != "ch1" || tree[0].index != -1 || len(tree[0].children) != 2 {
		t.Fatalf("ch1 = %+v children=%d", tree[0], len(tree[0].children))
	}
	if tree[0].children[0].name != "01.jpg" || tree[0].children[0].index != 0 {
		t.Fatalf("ch1/01 = %+v", tree[0].children[0])
	}
	if tree[1].name != "ch2" || tree[1].index != -1 || len(tree[1].children) != 1 {
		t.Fatalf("ch2 = %+v", tree[1])
	}
	if tree[1].children[0].index != 2 {
		t.Fatalf("ch2/01 index = %d", tree[1].children[0].index)
	}
}

func TestFlattenAlbumTree(t *testing.T) {
	tree := buildAlbumTree([]loader.FileInfo{
		{Name: "01.jpg", FilePath: "album/ch1/01.jpg"},
		{Name: "01.jpg", FilePath: "album/ch2/01.jpg"},
	})
	collapsed := flattenAlbumTree(tree, map[string]bool{})
	if len(collapsed) != 2 {
		t.Fatalf("collapsed rows = %d, want 2 folders", len(collapsed))
	}
	expanded := flattenAlbumTree(tree, map[string]bool{"ch1": true})
	if len(expanded) != 3 {
		t.Fatalf("expanded ch1 rows = %d, want 3", len(expanded))
	}
}
