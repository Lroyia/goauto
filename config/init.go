package config

import (
	"github.com/lroyia/goini"
)

type InitConfig struct {
	Conf map[string] string
	Dir map[string] string
	Branch map[string] string
	Script map[string] string
}

/**
 * 读取ini配置
 * @author lroyia
 * @since 2020年9月9日 10:03:19
 */
func ReadConfig(filePath string) (InitConfig, error) {
	// 读取ini配置
	conf, err := goini.Read(filePath)
	if err != nil{
		return InitConfig{}, err
	}
	dirs := conf.GetAllItemInSection("dir")
	branches := conf.GetAllItemInSection("branch")
	scripts := conf.GetAllItemInSection("script")
	confs := conf.GetAllItemInSection("conf")

	// 创建配置信息
	config := InitConfig{make(map[string]string), make(map[string]string), make(map[string]string), make(map[string]string)}
	for s := range confs {
		config.Conf[confs[s].Key] = confs[s].Value
	}
	for s := range dirs {
		config.Dir[dirs[s].Key] = dirs[s].Value
	}
	for s := range branches {
		config.Branch[branches[s].Key] = branches[s].Value
	}
	for s := range scripts {
		config.Script[scripts[s].Key] = scripts[s].Value
	}
	return config, nil
}
