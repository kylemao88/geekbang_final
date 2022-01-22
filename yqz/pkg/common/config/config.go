//
package config

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/config"
	"gopkg.in/yaml.v3"
)

const (
	// BakFlag is config flag in fileName
	BakFlag = "-bak-"
	// MaxBakCount is the max bak-config count
	MaxBakCount = 5
)

// InitConfigFromFile init config by local config file
func InitConfigFromFile(dir string, configName string, conf interface{}) error {
	configPath := dir + configName
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return errors.NewConfigError(errors.WithMsg("Read config file failed, config-file=[%s], err_msg=[%s]", configPath, err.Error()))
	}

	if err := yaml.Unmarshal(data, conf); err != nil {
		return errors.NewConfigError(errors.WithMsg("local config unmarshal failed, err_msg=[%s], content=[%s]", err.Error(), string(data)))
	}
	return nil
}

// GetRainbowConfig ...
func GetRainbowConfig(dir string, configName string, conf interface{}) error {
	// load config
	cfg, err := config.Load(configName, config.WithCodec("yaml"), config.WithProvider("rainbow"))
	if err != nil {
		logger.Error("error:%v", err)
		return err
	}
	if err := config.GetYAML(configName, conf); err != nil {
		logger.Error("error:%v", err)
		return err
	}
	logger.Info("get remote config success, config: %v", configName)
	// dump file
	err = DumpConfigFile(dir, configName, cfg)
	if err != nil {
		logger.Error("get remote config success, but dump file error: %v", err)
	}
	return nil
}

// DumpConfigFile ...
func DumpConfigFile(confDir string, configName string, cfg config.Config) error {
	configContent := cfg.Bytes()
	// check file exist
	if !isFileExist(confDir + configName) {
		// write directly
		return ioutil.WriteFile(confDir+configName, configContent, 0666)
	}

	// check exist config's content
	oldContent, err := ioutil.ReadFile(confDir + configName)
	if err != nil {
		logger.Error("ioutil.ReadFile failed, fileName: %v, err: %v", confDir+configName, err)
		return err
	}
	if string(oldContent) == string(configContent) {
		// config not update
		return nil
	}

	// list all exist config-file
	configList, err := ioutil.ReadDir(trpc.GlobalConfig().Server.ConfPath)
	if err != nil {
		logger.Error("ioutil.ReadDir failed, dirName: %v, err: %v", trpc.GlobalConfig().Server.ConfPath, err)
		return err
	}

	// traverse all config
	for idx := len(configList) - 1; idx >= 0; idx-- {
		info := configList[idx]
		if seq, err := traitsConfigFileInfo(configName, info.Name()); err != nil {
			// ignore the default config file, and it will not be used
			if info.Name() != "config.yaml" {
				logger.Info("traitsConfigFileInfo failed and skip, name: %v, info: %v", info.Name(), err)
				continue
			}
		} else if seq >= MaxBakCount-1 {
			continue
		} else {
			// rename config
			if err := os.Rename(confDir+info.Name(), confDir+configName+BakFlag+strconv.Itoa(seq+1)); err != nil {
				logger.Error("os.Rename failed, originName: %v, newName: %v, err: %v", confDir+info.Name(),
					confDir+configName+BakFlag+strconv.Itoa(seq+1), err)
				return err
			}
		}
	}

	// write newest config file
	return ioutil.WriteFile(confDir+configName, configContent, 0666)
}

// AddConfig is used to fetch remote-config from remote-config-center
func AddConfig(configName string, opts ...Option) error {
	// apply options
	opts = append(opts, withConfigName(configName))
	options := applyOptions(opts...)

	if _, _, err := updateLocalConfigCache(options); err != nil {
		return err
	}

	// check whether need to watch config-center
	if options.callback != nil {
		go watchRemoteConfig(options)
	}
	return nil
}

func applyOptions(opts ...Option) *options {
	options := &options{
		refreshInterval: 3 * time.Second,
		codec:           "yaml",
		provider:        "rainbow",
	}
	for o := range opts {
		opts[o](options)
	}
	return options
}

func watchRemoteConfig(opts *options) {
	notify := make(chan struct{})
	time.AfterFunc(opts.refreshInterval, func() { notify <- struct{}{} })
	for {
		select {
		case <-opts.callbackCtx.Done():
			logger.Info("exit watch remote config!")
			return
		case <-notify:
			// refresh config from remote-config-center
			if update, cfg, err := updateLocalConfigCache(opts); err != nil {
				logger.Error("updateLocalConfigCache failed, err: %v", err)
			} else if update {
				// call user-defined callback
				err := opts.callback(cfg.Bytes())
				if err != nil {
					logger.Error("callback fail: %d", err.Error())
				}
				logger.Info("receive replica update, result: [%v]", string(cfg.Bytes()))
			}

			time.AfterFunc(opts.refreshInterval, func() { notify <- struct{}{} })
		}
	}
}

func updateLocalConfigCache(options *options) (bool, config.Config, error) {
	configName := options.configName
	remoteConfigOptions := []config.LoadOption{
		config.WithProvider(options.provider),
		config.WithCodec(options.codec),
	}
	cfg, err := config.Load(configName, remoteConfigOptions...)
	if err != nil {
		logger.Error("config.Load failed, configName: %v, err: %v", configName, err)
		return false, nil, err
	}

	configContent := cfg.Bytes()
	confDir := trpc.GlobalConfig().Server.ConfPath
	// check file exist
	if !isFileExist(confDir + configName) {
		// write directly
		return true, cfg, ioutil.WriteFile(confDir+configName, configContent, 0666)
	}

	// check exist config's content
	oldContent, err := ioutil.ReadFile(confDir + configName)
	if err != nil {
		logger.Error("ioutil.ReadFile failed, fileName: %v, err: %v", confDir+configName, err)
		return true, nil, err
	}
	if string(oldContent) == string(configContent) {
		// config not update
		return false, cfg, nil
	}

	// list all exist config-file
	configList, err := ioutil.ReadDir(trpc.GlobalConfig().Server.ConfPath)
	if err != nil {
		logger.Error("ioutil.ReadDir failed, dirName: %v, err: %v", trpc.GlobalConfig().Server.ConfPath, err)
		return false, nil, err
	}

	// traverse all config
	for idx := len(configList) - 1; idx >= 0; idx-- {
		info := configList[idx]
		if seq, err := traitsConfigFileInfo(configName, info.Name()); err != nil {
			// ignore the default config file, and it will not be used
			if info.Name() != "config.yaml" {
				// logger.Error("traitsConfigFileInfo failed, name: %v, err: %v", info.Name(), err)
				continue
			}
		} else if seq >= MaxBakCount-1 {
			continue
		} else {
			// rename config
			if err := os.Rename(confDir+info.Name(), confDir+configName+BakFlag+strconv.Itoa(seq+1)); err != nil {
				logger.Error("os.Rename failed, originName: %v, newName: %v, err: %v", confDir+info.Name(),
					confDir+configName+BakFlag+strconv.Itoa(seq+1), err)
				return false, nil, err
			}
		}
	}

	// write newest config file
	return true, cfg, ioutil.WriteFile(confDir+configName, configContent, 0666)
}

func isFileExist(path string) bool {
	err := syscall.Access(path, syscall.F_OK)
	return !os.IsNotExist(err)
}

func traitsConfigFileInfo(configName string, file string) (int, error) {
	if configName == file {
		return 0, nil
	}

	index := strings.Index(file, BakFlag)
	if index < 0 {
		return 0, errors.NewInternalError(errors.WithMsg("config fileName format error, file: %v", file))
	}
	return strconv.Atoi(file[index+len(BakFlag):])
}
