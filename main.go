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
				err = build(key, dir, conf.Branch[key], conf.Script[key])
				if err != nil {
					log.Fatal(err.Error())
				}
			}
		} else {
			log.Fatal("构建失败，配置文件读取错误：" + err.Error())
		}
		time.Sleep(time.Duration(sleepTime) * time.Second)
	}
}

/**
 * 项目构建
 */
func build(key string, dir string, branch string, script string) error {
	// 执行结果定义
	var rs string

	// checkout 分支
	rs = callCmd(dir, "git", "checkout", branch)
	if !strings.Contains(rs, "Switched to branch") && !strings.Contains(rs, "Already on") && !strings.Contains(rs, "up to date") && rs != "" {
		return errors.New(key + " " + rs)
	}
	log.Println(key + " " + "git checkout " + branch + "  ---success")

	// 更新仓库
	rs = callCmd(dir, "git", "pull")
	if strings.Contains(rs, "Already") {
		log.Println(key + "  " + rs)
		return nil
	}
	log.Println(rs)

	// 关掉之前的应用
	pid := pidMap[key]
	if pid != "" {
		switch runtime.GOOS {
		case "windows":
			callCmd(dir, "taskkill", "/pid", pid, "-t", "-f")
		case "linux":
			callCmd(dir, "kill", "-9", pid)
		}
		if chMap[key] != nil {
			chMap[key] <- 1
		}
		log.Println("kill - " + pid)
	}

	// 拆分构建脚本
	script = strings.Trim(script, " ")
	if script == "" {
		return errors.New(key + " 缺少构建脚本")
	}
	scripts := strings.Split(script, "|")

	// 取最后一条脚本为运行脚本
	runScript := strings.Trim(scripts[len(scripts)-1:][0], " ")
	if runScript == "" {
		return errors.New(key + " 构建脚本有误")
	}
	// 取前面的脚本作为构建项目的脚本
	if len(scripts) > 1 {
		scripts = scripts[0 : len(scripts)-1]
		// 运行构建脚本
		for _, s := range scripts {
			log.Println(s + " --begin")
			callCmdStr(dir, s)
			log.Println(s + " --success")
		}
	}

	// 执行
	switch runtime.GOOS {
	case "windows":
		scripts = strings.SplitN(runScript, " ", 2)
		var args string
		binName := scripts[0]
		if len(scripts) > 1 {
			args = " " + scripts[1]
		}
		runScriptArray := strings.Split(runScript, " ")
		// 后台运行
		go callCmdNohup(dir, key, runScriptArray[0], runScriptArray[1:]...)
		time.Sleep(time.Duration(1) * time.Second)
		// 查询pid
		whereArgs := "CommandLine=\"" + binName + args + "\""
		rs = callCmd(dir, "wmic", "process", "where", whereArgs, "get", "ProcessId", "/value")
		rs = strings.Trim(strings.Trim(strings.Trim(rs, " "), "\n"), "\r")
		// 不是返回“没有可用实例”
		if !strings.Contains(rs, "没") && strings.Contains(rs, "=") {
			pidSplit := strings.Split(rs, "=")
			pidMap[key] = strings.ReplaceAll(strings.ReplaceAll(pidSplit[1], "\n", ""), "\r", "")
			log.Println("pid=" + pidMap[key])
		}
	case "linux":
		// 控制台输出文件名，用输入命令去空格，去“-”，去“.”
		runScript = strings.Trim(runScript, " ")
		log.Println(runScript + "  --begin")
		runScriptArray := strings.Split(runScript, " ")
		// 后台运行
		go callCmdNohup(dir, key, runScriptArray[0], runScriptArray[1:]...)
		time.Sleep(time.Duration(1) * time.Second)
		// 查询pid
		rs = callCmd(dir, "ps", "-ef")
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
	log.Println(runScript + " ---success")
	return nil
}

/**
 * 运行脚本
 */
func callCmdStr(dir string, cmd string) string {
	cmd = strings.Trim(cmd, " ")
	rs := strings.Split(cmd, " ")
	return callCmd(dir, rs[0], rs[1:]...)
}

/**
 * 调用命令
 * @author lroyia
 * @since 2020年9月9日 10:29:13
 */
func callCmd(dir string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	return out.String()
}

/**
 * 后台运行
 */
func callCmdNohup(dir string, key string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	err := cmd.Start()
	if err != nil {
		log.Fatal(err.Error())
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
