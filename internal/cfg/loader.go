package cfg

import (
  "log"
  "os"
  "sync"

  "github.com/fsnotify/fsnotify"
  "gopkg.in/yaml.v3"
)

type Loader struct {
  mu     sync.RWMutex
  config *Config
}

func NewLoader(path string) (*Loader, error) {
  l := &Loader{}
  if err := l.load(path); err != nil {
    return nil, err
  }

  watcher, _ := fsnotify.NewWatcher()
  watcher.Add(path)

  go func() {
    for ev := range watcher.Events {
      if ev.Op&fsnotify.Write == fsnotify.Write {
        log.Println("config.yaml changed → reloading")
        if err := l.load(path); err != nil {
          log.Println("reload failed:", err)
        }
      }
    }
  }()

  return l, nil
}

func (l *Loader) load(path string) error {
  b, err := os.ReadFile(path)
  if err != nil {
    return err
  }
  var c Config
  if err := yaml.Unmarshal(b, &c); err != nil {
    return err
  }

  l.mu.Lock()
  l.config = &c
  l.mu.Unlock()
  return nil
}

func (l *Loader) Get() *Config {
  l.mu.RLock()
  defer l.mu.RUnlock()
  return l.config
}
