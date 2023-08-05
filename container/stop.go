package container

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"

	"go-docker/common"
)

// 停止容器，修改容器状态
func StopContainer(containerName string) {
	info, err := getContainerInfo(containerName)
	if err != nil {
		logrus.Errorf("get container info, err: %v", err)
		return
	}
	if info.Pid != "" {
		pid, _ := strconv.Atoi(info.Pid)
		// 杀死进程
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			logrus.Errorf("stop container, pid: %d, err: %v", pid, err)
			return
		}
		isRun := isProcessRunning(pid)
		if isRun {
			err := syscall.Kill(pid, syscall.SIGKILL)
			if err != nil {
				logrus.Errorf("stop container (twice), pid: %d, err: %v", pid, err)
				return
			}
		}
		// 修改容器状态
		info.Status = common.Stop
		info.Pid = ""
		bs, _ := json.Marshal(info)
		fileName := path.Join(common.DefaultContainerInfoPath, containerName, common.ContainerInfoFileName)
		err := ioutil.WriteFile(fileName, bs, 0622)
		if err != nil {
			logrus.Errorf("write container config.json, err: %v", err)
		}
	}
}

func isProcessRunning(pid int) bool {
	// 尝试通过 PID 获取进程
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// 发送信号0来测试进程是否存在
	err = process.Signal(os.Signal(syscall.Signal(0)))
	return err == nil
}
