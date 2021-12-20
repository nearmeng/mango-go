package xres

import (
	"fmt"
	"io/ioutil"
	reflect "reflect"

	"github.com/nearmeng/mango-go/server_data/res"
	"google.golang.org/protobuf/proto"
)

type XresLoader struct {
}

func (l *XresLoader) loadRes(fileName string, fieldName string, m proto.Message) (res.Table, error) {
	mt := reflect.TypeOf(m).Elem()
	ft, ok := mt.FieldByName(fieldName)
	if !ok {
		return nil, fmt.Errorf("invalide field: %s", fieldName)
	}

	fileData, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("read file: %s", err)
	}

	blocks := XresloaderDatablocks{}
	if err := proto.Unmarshal(fileData, &blocks); err != nil {
		return nil, fmt.Errorf("proto unmarshal: %s", err)
	}

	tbl := res.NewResTable()
	for k, v := range blocks.DataBlock {
		mc := reflect.New(mt).Interface().(proto.Message)
		if err := proto.Unmarshal(v, mc); err != nil {
			return nil, fmt.Errorf("unmarshal %dth item: %s", k, err)
		}
		rv := reflect.ValueOf(mc).Elem()
		tbl.Insert(int32(rv.Field(ft.Index[0]).Int()), mc)
	}

	return tbl, nil
}

// LoadTables ...
func (l *XresLoader) LoadTables(cfg res.Config) (map[int32]res.Table, error) {
	return l.ReloadTables(cfg)
}

// ReloadTables ...
func (l *XresLoader) ReloadTables(config res.Config) (map[int32]res.Table, error) {

	tmpTables := map[int32]res.Table{}
	for _, cfg := range config.Tables {
		tbl, err := l.loadRes(config.ResPath+"/"+cfg.FileName, cfg.FieldName, cfg.Message)
		if err != nil {
			return nil, fmt.Errorf("load table %s: %s", cfg.FileName, err)
		}
		tmpTables[cfg.ID] = tbl

		if cfg.PostFunc != nil {
			cfg.PostFunc(cfg.ID, tbl)
		}
	}

	return tmpTables, nil
}

func init() {
	res.SetLoader(&XresLoader{})
}
