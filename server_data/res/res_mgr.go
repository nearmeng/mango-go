package res

import (
	"errors"
	"sync"

	"google.golang.org/protobuf/proto"
)

type loader interface {
	LoadTables(cfg Config) (map[int32]Table, error)
	ReloadTables(cfg Config) (map[int32]Table, error)
}

var (
	mutex  sync.RWMutex
	ld     loader = nil
	tables        = map[int32]Table{}
)

func SetLoader(l loader) {
	ld = l
}

func LoadRes(cfg Config) error {
	return ReloadRes(cfg)
}

func ReloadRes(cfg Config) error {
	if ld == nil {
		return errors.New("loader is nil")
	}

	tb, err := ld.LoadTables(cfg)
	if err != nil {
		return err
	}

	tables = tb
	return nil
}

func FindTable(tableID int32) Table {
	mutex.RLock()
	defer mutex.RUnlock()

	return tables[tableID]
}

func FindRes(tableID int32, key int32) proto.Message {
	mutex.RLock()
	defer mutex.RUnlock()

	if t, ok := tables[tableID]; ok {
		return t.Find(key)
	}
	return nil
}
