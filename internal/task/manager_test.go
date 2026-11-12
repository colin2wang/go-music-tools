package task

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func waitDone(t *testing.T, m *Manager) Progress {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if p := m.Status(); p.Done {
			return p
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("任务未在超时时间内结束")
	return Progress{}
}

func TestManagerProgressAndItemEvents(t *testing.T) {
	m := NewManager(nil)
	var mu sync.Mutex
	var items []ItemResult
	m.SetEmitter(func(event string, payload any) {
		if event != EventTaskItem {
			return
		}
		mu.Lock()
		items = append(items, payload.(ItemEvent).Item)
		mu.Unlock()
	})

	id, err := m.Start("download:song", 0, func(_ context.Context, r *Reporter) error {
		r.SetTotal(3)
		r.Stage("下载中")
		r.Item("第一首", "success", "已写入")
		r.ItemWithElapsed("第二首", "skipped", "已存在", 42)
		r.Item("第三首", "failed", "网络错误")
		return nil
	})
	if err != nil || id == "" {
		t.Fatalf("启动任务失败: id=%q err=%v", id, err)
	}

	p := waitDone(t, m)
	if p.TaskID != id || p.Total != 3 || p.Current != 3 {
		t.Fatalf("进度不一致: %+v", p)
	}
	if p.Success != 1 || p.Skipped != 1 || p.Failed != 1 {
		t.Fatalf("统计不一致: %+v", p)
	}
	if p.Stage != "已完成" {
		t.Fatalf("结束阶段错误: %q", p.Stage)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(items) != 3 {
		t.Fatalf("逐项事件数量 = %d, 期望 3", len(items))
	}
	if items[0].Status != "success" || items[2].Status != "failed" {
		t.Fatalf("逐项状态错误: %+v", items)
	}
	if items[1].ElapsedMS != 42 || items[1].Message != "已存在" {
		t.Fatalf("耗时与说明未随事件传递: %+v", items[1])
	}
}

func TestManagerCancel(t *testing.T) {
	m := NewManager(nil)
	started := make(chan struct{})
	if _, err := m.Start("library:lyrics", 1, func(ctx context.Context, _ *Reporter) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}
	<-started
	if !m.Cancel() {
		t.Fatal("运行中的任务应可取消")
	}
	p := waitDone(t, m)
	if !p.Cancelled || p.Stage != "已取消" {
		t.Fatalf("取消状态错误: %+v", p)
	}
	if m.Cancel() {
		t.Fatal("已结束的任务不应返回可取消")
	}
}

func TestManagerRejectsConcurrentTask(t *testing.T) {
	m := NewManager(nil)
	block := make(chan struct{})
	if _, err := m.Start("first", 1, func(context.Context, *Reporter) error {
		<-block
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start("second", 1, func(context.Context, *Reporter) error { return nil }); !errors.Is(err, ErrBusy) {
		t.Fatalf("期望 ErrBusy，实际 %v", err)
	}
	close(block)
	waitDone(t, m)
}
