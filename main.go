package main

import (
	"bytes"
	"errors"
	"goauto/config"
	"log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// 记录进程pid的map
var pidMap = make(map[string]string)

// 记录协程通信channel的map
var chMap = make(map[string]chan int)

func main() {
	for {
		// 先读取配置
		conf, err := config.ReadConfig("conf.ini")
		// 默认休眠时间5分钟
		sleepTime := 60 * 5
		// 配置没问题则进行构建
		if err == nil {
			// 读取休眠时间
			sleepInt, err := strconv.Atoi(conf.Conf["sleep"])
			if err == nil {
				sleepTime = sleepInt
			}
			// 构建项目
			for key, dir := range conf.Dir {
				err = build(key, dir, conf.Branch[key], conf.Build[key], conf.Run[key])
				if err != nil {
					log.Println(err.Error())
				}
			}
		} else {
			log.Println("构建失败，配置文件读取错误：" + err.Error())
		}
		time.Sleep(time.Duration(sleepTime) * time.Second)
		runtime.GC()
	}
}

/**
 * 项目构建
 */
func build(key string, dir string, branch string, build string, run string) error {
	// 执行结果定义
	var rs string

	// checkout 分支
	rs, _ = callCmd(dir, "git", "checkout", branch)
	if !strings.Contains(rs, "Switched to branch") && !strings.Contains(rs, "Already on") && !strings.Contains(rs, "up to date") && rs != "" {
		return errors.New(key + " " + rs)
	}
	log.Println(key + " " + "git checkout " + branch + "  ---success")

	// 更新仓库
	pid := pidMap[key]
	rs, _ = callCmd(dir, "git", "pull")
	if strings.Contains(rs, "Already") {
		log.Println(key + "  " + rs)
		return nil
	}
	log.Println(rs)

	// 关掉之前的应用
	if pid != "" {
		switch runtime.GOOS {
		case "windows":
			callCmd(dir, "taskkill", "/pid", pid, "-t", "-f")
		case "linux":
			callCmd(dir, "kill", "-15", pid)
		}
		if chMap[key] != nil {
			chMap[key] <- 1
		}
		log.Println("kill - " + pid)
	}

	// 拆分构建脚本
	build = strings.Trim(build, " ")
	if build == "" {
		return errors.New(key + " 缺少构建脚本")
	}
	builds := strings.Split(build, "|")

	// 运行构建脚本
	if len(builds) > 0 {
		for _, s := range builds {
			log.Println(s + " --begin")
			rs, err := callCmdStr(dir, s)
			log.Println(rs)
			if err != nil {
				return err
			}
			log.Println(s + " --success")
		}
	}

	// 运行执行脚本
	if strings.Trim(run, "") != "" {
		err := runAppScript(dir, key, run)
		if err != nil {
			return err
		}
	}

	_ = rs

	return nil
}

/**
 * 运行运行脚本
 */
func runAppScript(dir string, key string, runScript string) error {
	// 执行
	switch runtime.GOOS {
	case "windows":
		builds := strings.SplitN(runScript, " ", 2)
		var args string
		binName := builds[0]
		if len(builds) > 1 {
			args = " " + builds[1]
		}
		buildScriptArray := strings.Split(runScript, " ")
		// 后台运行
		go callCmdNohup(dir, key, buildScriptArray[0], buildScriptArray[1:]...)
		time.Sleep(time.Duration(1) * time.Second)
		// 查询pid
		whereArgs := "CommandLine=\"" + binName + args + "\""
		rs, err := callCmd(dir, "wmic", "process", "where", whereArgs, "get", "ProcessId", "/value")
		if err != nil {
			return err
		}
		rs = strings.Trim(strings.Trim(strings.Trim(rs, " "), "\n"), "\r")
		// 不是返回“没有可用实例”
		if !strings.Contains(rs, "没") && strings.Contains(rs, "=") {
			pidSplit := strings.Split(rs, "=")
			pidMap[key] = strings.ReplaceAll(strings.ReplaceAll(pidSplit[1], "\n", ""), "\r", "")
			log.Println("pid=" + pidMap[key])
		}
		_ = rs
	case "linux":
		// 控制台输出文件名，用输入命令去空格，去“-”，去“.”
		runScript = strings.Trim(runScript, " ")
		log.Println(runScript + "  --begin")
		buildScriptArray := strings.Split(runScript, " ")
		// 后台运行
		go callCmdNohup(dir, key, buildScriptArray[0], buildScriptArray[1:]...)
		time.Sleep(time.Duration(1) * time.Second)
		// 查询pid
		rs, err := callCmd(dir, "ps", "-ef")
		if err != nil {
			return err
		}
		rsLines := strings.Split(rs, "\n")
		target := ""
		for _, each := range rsLines {
			if strings.Contains(each, runScript) {
				target = each
				break
			}
		}
		if target == "" {
			return errors.New("could not find the pid")
		}
		pidTab := strings.Split(target, " ")
		pid := ""
		for _, each := range pidTab {
			_, err := strconv.ParseInt(each, 0, 32)
			if err == nil {
				pid = each
				break
			}
		}
		pidMap[key] = pid
		log.Println("pid=" + pid)
	}
	return nil
}

/**
 * 运行脚本
 */
func callCmdStr(dir string, cmd string) (string, error) {
	cmd = strings.Trim(cmd, " ")
	rs := strings.Split(cmd, " ")
	return callCmd(dir, rs[0], rs[1:]...)
}

/**
 * 调用命令
 * @author lroyia
 * @since 2020年9月9日 10:29:13
 */
func callCmd(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return out.String(), err
	}
	s := out.String()
	return s, nil
}

/**
 * 后台运行
 */
func callCmdNohup(dir string, key string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	err := cmd.Start()
	if err != nil {
		log.Println(err.Error())
		return
	}
	// 常见一个channel，用于关闭
	ch := make(chan int)
	chMap[key] = ch
	for {
		signal := <-ch
		if signal == 1 {
			delete(pidMap, key)
			delete(chMap, key)
			cmd.Wait()
			break
		}
	}
}
