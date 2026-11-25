package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type SnapshotMeta struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	CreatedAt       int64  `json:"created_at"`
	SizeBytes       int64  `json:"size_bytes"`
	HarnessVersion  string `json:"harness_version"`
	AppVersion      string `json:"app_version"`
	CustomCount     int    `json:"custom_count"`
	ConvCount       int    `json:"conv_count"`
	CompressionLv   int    `json:"compression_level"`
	FileName        string `json:"file_name"`
}

type SnapshotSummary struct {
	Items          []SnapshotMeta `json:"items"`
	TotalSizeBytes int64          `json:"total_size_bytes"`
}

type SnapshotProgressTask struct {
	Active   bool    `json:"active"`
	Action   string  `json:"action"` // create | restore
	Title    string  `json:"title"`
	Percent  float64 `json:"percent"`
}

type CreateSnapshotParams struct {
	Name             string `json:"name"`
	CompressionLevel int    `json:"compression_level"`
}

var (
	snapMu         sync.Mutex
	snapCancel     context.CancelFunc
	snapProgressMu sync.RWMutex
	curSnapProg    SnapshotProgressTask

	snapSubMu     sync.Mutex
	snapSubs      = map[chan struct{}]struct{}{}
	snapProgSubMu sync.Mutex
	snapProgSubs  = map[chan SnapshotProgressTask]struct{}{}
)

func SubscribeSnapshot(buf int) (<-chan struct{}, func()) {
	ch := make(chan struct{}, buf)
	snapSubMu.Lock()
	snapSubs[ch] = struct{}{}
	snapSubMu.Unlock()
	return ch, func() {
		snapSubMu.Lock()
		delete(snapSubs, ch)
		snapSubMu.Unlock()
	}
}

func notifySnapshotsChanged() {
	snapSubMu.Lock()
	for ch := range snapSubs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	snapSubMu.Unlock()
}

func SubscribeSnapshotProgress(buf int) (<-chan SnapshotProgressTask, func()) {
	ch := make(chan SnapshotProgressTask, buf)
	snapProgSubMu.Lock()
	snapProgSubs[ch] = struct{}{}
	snapProgSubMu.Unlock()
	return ch, func() {
		snapProgSubMu.Lock()
		delete(snapProgSubs, ch)
		snapProgSubMu.Unlock()
	}
}

func updateSnapProgress(task SnapshotProgressTask) {
	snapProgSubMu.Lock()
	snapProgressMu.Lock()
	curSnapProg = task
	snapProgressMu.Unlock()
	for ch := range snapProgSubs {
		select {
		case ch <- task:
		default:
		}
	}
	snapProgSubMu.Unlock()
}

func GetCurrentSnapshotProgress() SnapshotProgressTask {
	snapProgressMu.RLock()
	defer snapProgressMu.RUnlock()
	return curSnapProg
}

func ListSnapshots() (SnapshotSummary, error) {
	_ = os.MkdirAll(globalSnapshotsDir, 0755)
	entries, err := os.ReadDir(globalSnapshotsDir)
	if err != nil {
		return SnapshotSummary{}, err
	}
	items := make([]SnapshotMeta, 0)
	var totalBytes int64
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".meta.json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(globalSnapshotsDir, e.Name()))
		if err != nil {
			continue
		}
		var meta SnapshotMeta
		if err := json.Unmarshal(raw, &meta); err != nil {
			continue
		}
		tarPath := filepath.Join(globalSnapshotsDir, meta.FileName)
		if fi, err := os.Stat(tarPath); err == nil {
			meta.SizeBytes = fi.Size()
			totalBytes += fi.Size()
			items = append(items, meta)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return SnapshotSummary{
		Items:          items,
		TotalSizeBytes: totalBytes,
	}, nil
}

func CreateSnapshot(params CreateSnapshotParams) (SnapshotMeta, error) {
	snapMu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	snapCancel = cancel
	snapMu.Unlock()

	defer func() {
		snapMu.Lock()
		snapCancel = nil
		snapMu.Unlock()
		updateSnapProgress(SnapshotProgressTask{Active: false})
	}()

	lv := params.CompressionLevel
	if lv != 1 && lv != 6 && lv != 9 {
		lv = 1
	}

	id := generateShortID("snap")
	fileName := id + ".tar.gz"
	tarPath := filepath.Join(globalSnapshotsDir, fileName)
	metaPath := filepath.Join(globalSnapshotsDir, id+".meta.json")

	updateSnapProgress(SnapshotProgressTask{
		Active:  true,
		Action:  "create",
		Title:   fmt.Sprintf("正在打包快照「%s」…", params.Name),
		Percent: 10,
	})
	LogInfo("[快照] 开始创建快照「%s」(ID=%s, 压缩等级 Lv%d)", params.Name, id, lv)

	// Collect files to archive
	type fileEntry struct {
		absPath string
		relPath string
	}
	var files []fileEntry
	addDir := func(rootDir, prefix string) {
		_ = filepath.Walk(rootDir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(rootDir, p)
			if err != nil {
				return nil
			}
			files = append(files, fileEntry{
				absPath: p,
				relPath: filepath.ToSlash(filepath.Join(prefix, rel)),
			})
			return nil
		})
	}
	if _, err := os.Stat(configFilePath); err == nil {
		files = append(files, fileEntry{absPath: configFilePath, relPath: "config.json"})
	}
	addDir(globalDataDir, "data")
	addDir(globalGeminiHome, "gemini-home")

	outFile, err := os.Create(tarPath)
	if err != nil {
		LogError("[快照] 创建归档文件失败: %s", err)
		return SnapshotMeta{}, err
	}
	gzWriter, err := gzip.NewWriterLevel(outFile, lv)
	if err != nil {
		_ = outFile.Close()
		_ = os.Remove(tarPath)
		return SnapshotMeta{}, err
	}
	tw := tar.NewWriter(gzWriter)

	total := len(files)
	if total == 0 {
		total = 1
	}
	for idx, fe := range files {
		select {
		case <-ctx.Done():
			_ = tw.Close()
			_ = gzWriter.Close()
			_ = outFile.Close()
			_ = os.Remove(tarPath)
			LogWarning("[快照] 快照创建任务「%s」已被用户取消", params.Name)
			return SnapshotMeta{}, fmt.Errorf("已取消")
		default:
		}
		info, err := os.Stat(fe.absPath)
		if err != nil {
			continue
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			continue
		}
		hdr.Name = fe.relPath
		if err := tw.WriteHeader(hdr); err != nil {
			continue
		}
		if f, err := os.Open(fe.absPath); err == nil {
			_, _ = io.Copy(tw, f)
			_ = f.Close()
		}
		pct := 15 + float64(idx+1)/float64(total)*80
		updateSnapProgress(SnapshotProgressTask{
			Active:  true,
			Action:  "create",
			Title:   fmt.Sprintf("正在打包快照「%s」(%d/%d)…", params.Name, idx+1, total),
			Percent: pct,
		})
	}

	_ = tw.Close()
	_ = gzWriter.Close()
	_ = outFile.Close()

	fi, _ := os.Stat(tarPath)
	sizeBytes := int64(0)
	if fi != nil {
		sizeBytes = fi.Size()
	}
	cs := GetCustomizationsSummary()
	meta := SnapshotMeta{
		ID:             id,
		Name:           params.Name,
		CreatedAt:      time.Now().Unix(),
		SizeBytes:      sizeBytes,
		HarnessVersion: GetConfig().Version,
		AppVersion:     globalAppVer,
		CustomCount:    cs.SkillsCount + cs.MCPCount + cs.PluginsCount + cs.RulesCount + cs.HooksCount,
		ConvCount:      len(ListConversations(true)),
		CompressionLv:  lv,
		FileName:       fileName,
	}
	if b, err := json.MarshalIndent(meta, "", "  "); err == nil {
		_ = os.WriteFile(metaPath, b, 0644)
	}

	updateSnapProgress(SnapshotProgressTask{
		Active:  true,
		Action:  "create",
		Title:   fmt.Sprintf("快照「%s」创建完成", params.Name),
		Percent: 100,
	})
	time.Sleep(300 * time.Millisecond)
	LogInfo("[快照] 快照「%s」创建完成 (大小 %.1f KB)", params.Name, float64(sizeBytes)/1024.0)
	notifySnapshotsChanged()
	return meta, nil
}

func CancelSnapshotCreation() error {
	snapMu.Lock()
	cancel := snapCancel
	snapMu.Unlock()
	if cancel == nil {
		return fmt.Errorf("当前没有正在进行的快照任务")
	}
	cancel()
	return nil
}

func RestoreSnapshot(id string) error {
	metaPath := filepath.Join(globalSnapshotsDir, filepath.Base(id)+".meta.json")
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("快照不存在: %s", id)
	}
	var meta SnapshotMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return err
	}
	tarPath := filepath.Join(globalSnapshotsDir, meta.FileName)
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("快照归档文件缺失: %w", err)
	}
	defer f.Close()

	defer func() {
		updateSnapProgress(SnapshotProgressTask{Active: false})
	}()
	updateSnapProgress(SnapshotProgressTask{
		Active:  true,
		Action:  "restore",
		Title:   fmt.Sprintf("正在还原快照「%s」…", meta.Name),
		Percent: 30,
	})
	LogInfo("[快照] 开始还原快照「%s」(%s)", meta.Name, id)

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		cleanRel := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanRel, "..") || filepath.IsAbs(cleanRel) {
			continue
		}
		target := filepath.Join(globalPkgVar, cleanRel)
		if hdr.Typeflag == tar.TypeDir {
			_ = os.MkdirAll(target, 0755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(target), 0755)
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			continue
		}
		_, _ = io.Copy(out, tr)
		_ = out.Close()
	}

	updateSnapProgress(SnapshotProgressTask{
		Active:  true,
		Action:  "restore",
		Title:   fmt.Sprintf("快照「%s」还原完成，正在重新加载配置…", meta.Name),
		Percent: 95,
	})
	InitConfig()
	InitAppEnv()
	notifyConversationsChanged()
	notifyWorkspaceChanged()
	state.Poke()
	time.Sleep(300 * time.Millisecond)
	LogInfo("[快照] 快照「%s」还原完成", meta.Name)
	return nil
}

func DeleteSnapshot(id string) error {
	cleanID := filepath.Base(id)
	metaPath := filepath.Join(globalSnapshotsDir, cleanID+".meta.json")
	tarPath := filepath.Join(globalSnapshotsDir, cleanID+".tar.gz")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return fmt.Errorf("快照不存在: %s", id)
	}
	_ = os.Remove(metaPath)
	_ = os.Remove(tarPath)
	LogInfo("[快照] 已删除快照 %s", cleanID)
	notifySnapshotsChanged()
	return nil
}
