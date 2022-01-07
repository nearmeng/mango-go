package keep

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"git.code.oa.com/rainbow/golang-sdk/log"
	"git.code.oa.com/rainbow/golang-sdk/types"
	"git.code.oa.com/rainbow/golang-sdk/utils"
	v3 "git.code.oa.com/rainbow/proto/api/configv3"
	"github.com/pkg/errors"
)

var (
	defaultDir = "/tmp/"
)

// getDir default dir
func (fc *FullCache) getDir() string {
	dir := filepath.Dir(fc.opts.FileCachePath)
	if dir == "" {
		dir = defaultDir
	}
	return dir
}

// SetOpts set opts
func (fc *FullCache) SetOpts(opts types.InitOptions) {
	fc.opts = opts
}

func (fc *FullCache) isSameFile(key string, m *FileMeta) bool {
	iv, ok := fc.fileCache.Load(key)
	if !ok {
		return false
	}
	src := iv.(*FileMeta)
	if src.MD5 == m.MD5 {
		return true
	}
	return false
}

// TypeFile 处理文件类型
func (fc *FullCache) TypeFile(opts types.GetOptions, fdl *v3.FileDataList) error {
	if fdl == nil {
		return nil
	}
	for _, v := range fdl.FileDatas {
		meta := &FileMeta{}

		dec := json.NewDecoder(bytes.NewReader(utils.StringToBytes(v.GetMetadata())))
		dec.UseNumber()
		err := dec.Decode(meta)
		if err != nil {
			log.Errorf("err=%s", err.Error())
			return err
		}

		key := opts.AppID + opts.EnvName + opts.Group + meta.Name
		if fc.isSameFile(key, meta) {
			continue
		}
		err = fc.DownloadFile(meta)
		if err != nil {
			log.Errorf("DownloadFile [%s] err=[%s]", key, err.Error())
			continue
		}
		fc.fileCache.Store(key, meta)
	}

	return nil
}

// LoadFileMeta  加载文件内容
func (fc *FullCache) LoadFileMeta(opts types.GetOptions, fdl *v3.FileDataList) FileMetaList {
	fdml := make(FileMetaList, 0)
	if fdl == nil {
		return fdml
	}

	for _, v := range fdl.FileDatas {
		meta := &FileMeta{}

		dec := json.NewDecoder(bytes.NewReader(utils.StringToBytes(v.GetMetadata())))
		dec.UseNumber()
		err := dec.Decode(meta)
		if err != nil {
			log.Errorf("err=%s", err.Error())
			return fdml
		}

		key := opts.AppID + opts.EnvName + opts.Group + meta.Name
		if v, ok := fc.fileCache.Load(key); ok {
			fdml = append(fdml, v.(*FileMeta))
		} else { // 不存在加载一次
			err = fc.DownloadFile(meta)
			if err != nil {
				log.Errorf("DownloadFile [%s] err=[%s]", key, err.Error())
				continue
			}
			fc.fileCache.Store(key, meta)
			fdml = append(fdml, meta)
		}
	}
	return fdml
}

// DownloadFile 下载文件
func (fc *FullCache) DownloadFile(m *FileMeta) error {
	fullFileName := fc.getDir() + "/" + m.Name
	path, _ := filepath.Split(fullFileName)

	// get the data
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
		},
	}
	req, err := http.NewRequest("GET", m.URL+"?versionId="+m.FileVersion, nil)
	if err != nil {
		return errors.Wrap(err, "http download NewRequest")
	}
	if m.SecrectKey != "" {
		req.Header.Add("x-cos-server-side-encryption-customer-algorithm", "AES256")
		req.Header.Add("x-cos-server-side-encryption-customer-key", m.SecrectKey)
		req.Header.Add("x-cos-server-side-encryption-customer-key-MD5", m.SecretKeyMD5)
	}
	req.Close = true
	ctx, cancelFunc := context.WithTimeout(context.Background(), fc.opts.TimeoutDownload)
	req = req.WithContext(ctx)
	defer cancelFunc()

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "http download ")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("StatusCode=%d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "http read body")
	}
	m.Content = utils.BytesToString(body)
	err = fc.CreateDir(path)
	if err != nil {
		return errors.Wrap(err, "CreateDir")
	}

	if fc.opts.IsUsingFileCache {
		err = ioutil.WriteFile(fullFileName, body, 0664)
		if err != nil {
			return errors.Wrap(err, "WriteFile")
		}
	}
	return nil
}

// CreateDir 目录不存在则创建
func (fc *FullCache) CreateDir(dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		return err
	}
	err = os.MkdirAll(dir, 0777)
	return err
}
