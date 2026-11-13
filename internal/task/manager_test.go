package task

import (
	"context"
	"strings"
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

// waitEntry 轮询注册表直到指定任务满足期望状态。
func waitEntry(t *testing.T, m *Manager, id string, match func(TaskEntry) bool) TaskEntry {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range m.Board().Tasks {
			if e.TaskID == id && match(e) {
				return e
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("任务 %s 未在超时时间内达到期望状态", id)
	return TaskEntry{}
}

func entryOf(t *testing.T, m *Manager, id string) TaskEntry {
	t.Helper()
	for _, e := range m.Board().Tasks {
		if e.TaskID == id {
			return e
		}
	}
	t.Fatalf("注册表里找不到任务 %s", id)
	return TaskEntry{}
}

func TestManagerProgressAndItemEvents(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
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
	defer m.Close()
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

func TestManagerEnqueuesConcurrentTask(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
	block := make(chan struct{})
	firstStarted := make(chan struct{})
	if _, err := m.Start("first", 1, func(context.Context, *Reporter) error {
		close(firstStarted)
		<-block
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	<-firstStarted

	// 运行中提交：进入队列而非报错
	id2, err := m.Start("second", 1, func(context.Context, *Reporter) error { return nil })
	if err != nil || id2 == "" {
		t.Fatalf("排队提交失败: id=%q err=%v", id2, err)
	}
	queued := m.Queued()
	if len(queued) != 1 || queued[0].TaskID != id2 || queued[0].Kind != "second" {
		t.Fatalf("队列状态错误: %+v", queued)
	}

	// 移除排队任务
	if !m.RemoveQueued(id2) || len(m.Queued()) != 0 {
		t.Fatal("移除排队任务失败")
	}

	// 清空后重新入队，当前任务结束后应自动派发
	id3, err := m.Start("third", 1, func(_ context.Context, r *Reporter) error {
		r.Item("第三", "success", "")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	close(block)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if p := m.Status(); p.Done && p.TaskID == id3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if p := m.Status(); !p.Done || p.TaskID != id3 || p.Success != 1 {
		t.Fatalf("排队任务未被派发或结果错误: %+v", p)
	}
}

func TestManagerClearQueue(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
	block := make(chan struct{})
	if _, err := m.Start("first", 1, func(context.Context, *Reporter) error {
		<-block
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := m.Start("queued", 1, func(context.Context, *Reporter) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if n := m.ClearQueue(); n != 3 || len(m.Queued()) != 0 {
		t.Fatalf("清空队列数量错误: n=%d", n)
	}
	close(block)
}

func TestManagerLanesRunIndependently(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
	workStarted := make(chan struct{})
	ooStarted := make(chan struct{})
	release := make(chan struct{})
	if _, err := m.Start("download:song", 1, func(context.Context, *Reporter) error {
		close(workStarted)
		<-release
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	<-workStarted
	if _, err := m.Start("oomusic:single", 1, func(context.Context, *Reporter) error {
		close(ooStarted)
		<-release
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ooStarted:
	case <-time.After(time.Second):
		t.Fatal("OO Music 通道不应被工作台任务阻塞")
	}
	board := m.Board()
	running := 0
	for _, e := range board.Tasks {
		if e.Status == TaskStatusRunning {
			running++
		}
	}
	if running != 2 {
		t.Fatalf("期望两条通道各运行一个任务，实际 %d", running)
	}
	if len(board.Lanes) != 2 || board.Lanes[0].Running == "" || board.Lanes[1].Running == "" {
		t.Fatalf("通道状态不完整: %+v", board.Lanes)
	}
	close(release)
}

func TestManagerSameLaneStaysSerial(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
	release := make(chan struct{})
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	if _, err := m.Start("download:song", 1, func(context.Context, *Reporter) error {
		close(firstStarted)
		<-release
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	<-firstStarted
	secondID, err := m.Start("download:album", 1, func(context.Context, *Reporter) error {
		close(secondStarted)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-secondStarted:
		t.Fatal("同通道第二个任务不应提前开始")
	case <-time.After(80 * time.Millisecond):
	}
	if e := entryOf(t, m, secondID); e.Status != TaskStatusQueued {
		t.Fatalf("排队任务状态错误: %+v", e)
	}
	close(release)
	select {
	case <-secondStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("当前任务结束后排队任务未被自动派发")
	}
}

func TestManagerTaskEntryLifecycle(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
	release := make(chan struct{})
	started := make(chan struct{})
	id, err := m.Start("download:playlist", 2, func(_ context.Context, r *Reporter) error {
		close(started)
		r.Item("第一首", "failed", "网络错误")
		<-release
		r.Item("第二首", "success", "已写入")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	e := waitEntry(t, m, id, func(e TaskEntry) bool { return e.Failed == 1 })
	if e.Lane != LaneWorkbench || e.Status != TaskStatusRunning {
		t.Fatalf("运行中注册表状态错误: %+v", e)
	}
	if e.EnqueuedAt == 0 || e.StartedAt == 0 || e.FinishedAt != 0 {
		t.Fatalf("时间戳错误: %+v", e)
	}
	if e.ItemCount != 1 || len(e.Items) != 1 || e.Items[0].Status != "failed" {
		t.Fatalf("子项未写入注册表: %+v", e)
	}
	close(release)
	e = waitEntry(t, m, id, func(e TaskEntry) bool { return e.Status != TaskStatusRunning })
	if e.Status != TaskStatusDone || e.FinishedAt < e.StartedAt || e.ItemCount != 2 {
		t.Fatalf("结束状态错误: %+v", e)
	}
}

func TestManagerCancelByIDKeepsOtherLane(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
	workStarted := make(chan struct{})
	ooStarted := make(chan struct{})
	workID, err := m.Start("download:song", 1, func(ctx context.Context, _ *Reporter) error {
		close(workStarted)
		<-ctx.Done()
		return ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	<-workStarted
	ooID, err := m.Start("oomusic:single", 1, func(ctx context.Context, _ *Reporter) error {
		close(ooStarted)
		<-ctx.Done()
		return ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	<-ooStarted
	if !m.CancelByID(workID) {
		t.Fatal("按 ID 取消应成功")
	}
	waitEntry(t, m, workID, func(e TaskEntry) bool { return e.Status == TaskStatusCancelled })
	if e := entryOf(t, m, ooID); e.Status != TaskStatusRunning {
		t.Fatalf("另一通道不应被取消: %+v", e)
	}
	m.Cancel()
}

func TestManagerRetryFailedItems(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
	factory := func(failed []ItemResult) (TaskFunc, error) {
		return func(_ context.Context, r *Reporter) error {
			for _, item := range failed {
				r.Item(item.Name, "success", "重试完成")
			}
			return nil
		}, nil
	}
	id, err := m.Start("download:songs", 0, func(_ context.Context, r *Reporter) error {
		r.Item("song-1", "failed", "网络错误")
		r.Item("song-2", "success", "已写入")
		return nil
	}, WithRetry(factory))
	if err != nil {
		t.Fatal(err)
	}
	waitEntry(t, m, id, func(e TaskEntry) bool { return e.Status == TaskStatusDone })
	if !entryOf(t, m, id).CanRetry {
		t.Fatal("注入重试工厂后 CanRetry 应为 true")
	}
	retryID, err := m.RetryFailedItems(id)
	if err != nil {
		t.Fatal(err)
	}
	if retryID == id {
		t.Fatal("重试应使用新的 task_id")
	}
	e := waitEntry(t, m, retryID, func(e TaskEntry) bool { return e.Status == TaskStatusDone })
	if e.Total != 1 || e.Success != 1 || e.ItemCount != 1 {
		t.Fatalf("重试任务只应包含失败子项: %+v", e)
	}
	if !strings.HasSuffix(e.Kind, retryKindSuffix) || e.Lane != LaneWorkbench {
		t.Fatalf("重试任务 kind/lane 错误: %+v", e)
	}
}

func TestManagerRetryGuards(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
	libraryID, err := m.Start("library:lyrics", 0, func(_ context.Context, r *Reporter) error {
		r.Item("folder", "success", "已处理")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	waitEntry(t, m, libraryID, func(e TaskEntry) bool { return e.Status == TaskStatusDone })
	if entryOf(t, m, libraryID).CanRetry {
		t.Fatal("库任务未注入工厂，CanRetry 应为 false")
	}
	if _, err := m.RetryFailedItems(libraryID); err == nil {
		t.Fatal("不支持重试的任务应返回错误")
	}

	noFailID, err := m.Start("download:songs", 0, func(_ context.Context, r *Reporter) error {
		r.Item("song-1", "success", "已写入")
		return nil
	}, WithRetry(func([]ItemResult) (TaskFunc, error) {
		return func(context.Context, *Reporter) error { return nil }, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	waitEntry(t, m, noFailID, func(e TaskEntry) bool { return e.Status == TaskStatusDone })
	if _, err := m.RetryFailedItems(noFailID); err == nil {
		t.Fatal("没有失败子项时应返回错误")
	}
}

func TestManagerQueueRemovalMarksEntry(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
	release := make(chan struct{})
	started := make(chan struct{})
	firstID, err := m.Start("download:song", 1, func(context.Context, *Reporter) error {
		close(started)
		<-release
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	queuedID, err := m.Start("download:album", 0, func(context.Context, *Reporter) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	ooStarted := make(chan struct{})
	ooID, err := m.Start("oomusic:batch", 0, func(ctx context.Context, _ *Reporter) error {
		close(ooStarted)
		<-ctx.Done()
		return ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	<-ooStarted
	// OO 通道空闲，直接运行；工作台通道的任务仍在排队。
	if e := entryOf(t, m, ooID); e.Status != TaskStatusRunning {
		t.Fatalf("OO 通道任务不应排队: %+v", e)
	}
	if !m.RemoveQueued(queuedID) {
		t.Fatal("移除跨通道排队任务应成功")
	}
	if e := entryOf(t, m, queuedID); e.Status != TaskStatusRemoved {
		t.Fatalf("被移除任务应标记 removed: %+v", e)
	}
	if e := entryOf(t, m, firstID); e.Status != TaskStatusRunning {
		t.Fatalf("运行中任务不应被移除影响: %+v", e)
	}
	if n := m.ClearQueue(); n != 0 {
		t.Fatalf("此时不应还有排队任务: n=%d", n)
	}
	close(release)
	m.Cancel()
}

func TestManagerBoardKeepsAtMost50Tasks(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
	for i := 0; i < maxTaskEntries+12; i++ {
		id, err := m.Start("download:song", 0, func(context.Context, *Reporter) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		waitEntry(t, m, id, func(e TaskEntry) bool {
			return e.Status != TaskStatusRunning && e.Status != TaskStatusQueued
		})
	}
	if got := len(m.Board().Tasks); got > maxTaskEntries {
		t.Fatalf("注册表数量 %d 超过上限 %d", got, maxTaskEntries)
	}
}
