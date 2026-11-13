package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigIsValid(t *testing.T) {
	if errs := Default().Validate(); len(errs) != 0 {
		t.Fatalf("默认配置不合法: %v", errs)
	}
}

func TestValidationCatchesBadFields(t *testing.T) {
	c := Default()
	c.API.BaseURL = "not-a-url"
	c.Download.DefaultQuality = "ultra"
	c.Download.IndexFormat = "abc"
	c.Filename.Pattern = "{unknown}"
	c.Logging.Level = "verbose"

	errs := c.Validate()
	for _, key := range []string{
		"api.base_url", "download.default_quality", "download.index_format",
		"filename.pattern", "logging.level",
	} {
		if _, ok := errs[key]; !ok {
			t.Errorf("未校验出字段 %s，实际结果: %v", key, errs)
		}
	}
}

func TestStoreCreatesDefaultAndReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	store := NewStore(path)

	if err := store.Load(); err != nil {
		t.Fatalf("首次加载失败: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("未生成默认配置文件: %v", err)
	}

	c := store.Get()
	c.Download.DefaultQuality = "hires"
	c.Network.MaxRetries = 4
	if err := store.Save(c); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Fatal("临时文件未被清理")
	}

	other := NewStore(path)
	if err := other.Load(); err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}
	got := other.Get()
	if got.Download.DefaultQuality != "hires" || got.Network.MaxRetries != 4 {
		t.Fatalf("重载后的值不正确: %+v", got)
	}
	if got.Metadata.WriteCover != Default().Metadata.WriteCover {
		t.Fatal("默认值合并丢失了未修改字段")
	}
}

func TestSaveRejectsInvalidConfig(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "config.yml"))
	c := Default()
	c.Network.Timeout = -1
	err := store.Save(c)
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("期望 ValidationError，实际 %T: %v", err, err)
	}
	if _, exists := ve.Errors["network.timeout"]; !exists {
		t.Fatalf("缺少字段级错误: %v", ve.Errors)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	store := NewStore(path)
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}
	c := store.Get()
	c.Download.DefaultQuality = "hires"
	c.Advanced.OOMusicDefaultQuality = "HQ"
	if err := store.Save(c); err != nil {
		t.Fatal(err)
	}

	backup := filepath.Join(dir, "backup", "config-backup.yaml")
	if err := store.Export(backup); err != nil {
		t.Fatalf("导出失败: %v", err)
	}
	if _, err := os.Stat(backup); err != nil {
		t.Fatalf("导出文件不存在: %v", err)
	}
	if _, err := os.Stat(backup + ".tmp"); err == nil {
		t.Fatal("导出的临时文件未被清理")
	}

	changed := store.Get()
	changed.Download.DefaultQuality = "standard"
	if err := store.Save(changed); err != nil {
		t.Fatal(err)
	}
	var notified Config
	store.SetOnChange(func(next Config) { notified = next })
	if err := store.Import(backup); err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	if got := store.Get(); got.Download.DefaultQuality != "hires" || got.Advanced.OOMusicDefaultQuality != "HQ" {
		t.Fatalf("导入后配置未恢复: %+v", got)
	}
	if notified.Download.DefaultQuality != "hires" {
		t.Fatal("导入未触发配置变更广播")
	}
	other := NewStore(path)
	if err := other.Load(); err != nil {
		t.Fatal(err)
	}
	if other.Get().Download.DefaultQuality != "hires" {
		t.Fatal("导入结果未原子写回 config.yml")
	}
}

func TestImportRejectsInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "config.yml"))
	if err := store.Load(); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("network:\n  timeout: -5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := store.Import(bad)
	if _, ok := err.(*ValidationError); !ok {
		t.Fatalf("期望 ValidationError，实际 %T: %v", err, err)
	}
	if store.Get().Network.Timeout != Default().Network.Timeout {
		t.Fatal("导入失败不应污染当前生效配置")
	}
}

func TestSchemaCoversAllFields(t *testing.T) {
	c := Default()
	groups := Schema(c)
	if len(groups) != 7 {
		t.Fatalf("分组数量不符: %d", len(groups))
	}
	count := 0
	for _, g := range groups {
		for _, f := range g.Fields {
			count++
			if f.Value == nil {
				t.Errorf("字段 %s.%s 反射取值为空", g.Key, f.Key)
			}
			if f.Default == nil {
				t.Errorf("字段 %s.%s 缺少默认值", g.Key, f.Key)
			}
			if f.Description == "" {
				t.Errorf("字段 %s.%s 缺少说明", g.Key, f.Key)
			}
		}
	}
	if count < 40 {
		t.Fatalf("schema 字段过少: %d", count)
	}
}
