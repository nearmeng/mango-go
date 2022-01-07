package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"git.code.oa.com/trpc-go/trpc-go/log"
	"github.com/fsnotify/fsnotify"
)

func init() {
	RegisterProvider(newFileProvider())
}

func newFileProvider() *FileProvider {
	fp := &FileProvider{
		cb:              make(chan ProviderCallback),
		disabledWatcher: true,
		cache:           make(map[string]string),
		modtime:         make(map[string]int64),
	}
	if watcher, err := fsnotify.NewWatcher(); err == nil {
		fp.disabledWatcher = false
		fp.watcher = watcher
		go fp.run()
	}
	return fp
}

// FileProvider 从文件系统拉取文件内容
type FileProvider struct {
	disabledWatcher bool
	watcher         *fsnotify.Watcher
	cb              chan ProviderCallback
	cache           map[string]string
	modtime         map[string]int64
	mu              sync.RWMutex
}

// Name Provider名字
func (*FileProvider) Name() string {
	return "file"
}

// Read 读取指定文件
func (fp *FileProvider) Read(path string) ([]byte, error) {
	if !fp.disabledWatcher {
		if err := fp.watcher.Add(filepath.Dir(path)); err != nil {
			return nil, err
		}
		fp.mu.Lock()
		fp.cache[filepath.Clean(path)] = path
		fp.mu.Unlock()
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Tracef("Failed to read file %v", err)
		return nil, err
	}
	return data, nil
}

// Watch 注册文件变化处理函数
func (fp *FileProvider) Watch(cb ProviderCallback) {
	if !fp.disabledWatcher {
		fp.cb <- cb
	}
}

func (fp *FileProvider) run() {
	fn := make([]ProviderCallback, 0)
	for {
		select {
		case i := <-fp.cb:
			fn = append(fn, i)
		case e := <-fp.watcher.Events:
			if t, ok := fp.isModified(e); ok {
				fp.trigger(e, t, fn)
			}
		}
	}
}

func (fp *FileProvider) isModified(e fsnotify.Event) (int64, bool) {
	if e.Op&fsnotify.Write != fsnotify.Write {
		return 0, false
	}
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	if _, ok := fp.cache[filepath.Clean(e.Name)]; !ok {
		return 0, false
	}
	fi, err := os.Stat(e.Name)
	if err != nil {
		return 0, false
	}
	if fi.ModTime().Unix() > fp.modtime[e.Name] {
		return fi.ModTime().Unix(), true
	}
	return 0, false
}

func (fp *FileProvider) trigger(e fsnotify.Event, t int64, fn []ProviderCallback) {
	data, err := ioutil.ReadFile(e.Name)
	if err != nil {
		return
	}
	fp.mu.Lock()
	path := fp.cache[filepath.Clean(e.Name)]
	fp.modtime[e.Name] = t
	fp.mu.Unlock()
	for _, f := range fn {
		go f(path, data)
	}
}
