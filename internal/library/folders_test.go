package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go-music-tools/internal/model"
	"go-music-tools/internal/task"
)

func makeAlbumFolders(t *testing.T, names ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range names {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestListFoldersSkipsFilesAndHidden(t *testing.T) {
	root := makeAlbumFolders(t, "Beta", "alpha", ".hidden")
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	folders, err := ListFolders(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 2 {
		t.Fatalf("目录数 = %d, 期望 2: %v", len(folders), folders)
	}
	if filepath.Base(folders[0]) != "alpha" || filepath.Base(folders[1]) != "Beta" {
		t.Fatalf("未按名称大小写无关排序: %v", folders)
	}
}

func waitManagerDone(t *testing.T, m *task.Manager) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if m.Status().Done {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("修补任务未在超时时间内结束")
}

func runFolders(t *testing.T, root string, continueOnError bool) ([]model.FolderResult, []task.ItemResult) {
	t.Helper()
	m := task.NewManager(nil)
	var mu sync.Mutex
	items := []task.ItemResult{}
	m.SetEmitter(func(event string, payload any) {
		if event != task.EventTaskItem {
			return
		}
		mu.Lock()
		items = append(items, payload.(task.ItemEvent).Item)
		mu.Unlock()
	})

	var results []model.FolderResult
	_, err := m.Start("library:lyrics", 0, func(ctx context.Context, r *task.Reporter) error {
		var runErr error
		results, runErr = ProcessFolders(ctx, root, func(_ context.Context, folder string) (string, error) {
			if filepath.Base(folder) == "02" {
				return "", errors.New("模拟修补失败")
			}
			return "修补完成", nil
		}, Options{ContinueOnError: continueOnError, Reporter: r})
		return runErr
	})
	if err != nil {
		t.Fatal(err)
	}
	waitManagerDone(t, m)
	mu.Lock()
	defer mu.Unlock()
	return results, append([]task.ItemResult(nil), items...)
}

func TestProcessFoldersContinuesOnError(t *testing.T) {
	root := makeAlbumFolders(t, "01", "02", "03")
	results, items := runFolders(t, root, true)
	if len(results) != 3 {
		t.Fatalf("应处理全部 3 个目录，实际 %d", len(results))
	}
	if !results[0].Success || results[1].Success || !results[2].Success {
		t.Fatalf("结果状态错误: %+v", results)
	}
	if results[1].Message != "模拟修补失败" {
		t.Fatalf("失败原因未记录: %q", results[1].Message)
	}
	if len(items) != 3 || items[1].Status != "failed" || items[1].Message != "模拟修补失败" {
		t.Fatalf("逐目录事件错误: %+v", items)
	}
	if items[0].Name != "01" || items[0].Status != "success" {
		t.Fatalf("首个目录事件错误: %+v", items[0])
	}
}

func TestProcessFoldersStopsOnError(t *testing.T) {
	root := makeAlbumFolders(t, "01", "02", "03")
	results, items := runFolders(t, root, false)
	if len(results) != 2 || len(items) != 2 {
		t.Fatalf("应在第二个目录后中断: results=%d items=%d", len(results), len(items))
	}
	if results[1].Success {
		t.Fatal("第二个目录应为失败")
	}
}
