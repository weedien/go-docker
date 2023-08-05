package container

import (
	"github.com/sirupsen/logrus"

	"go-docker/common"
)

// 删除容器
func RemoveContainer(containerName string) {
	info, err := getContainerInfo(containerName)
	if err != nil {
		logrus.Errorf("get container info, err: %v", err)
		return
	}
	// 只能删除停止状态的容器
	if info.Status != common.Stop {
		logrus.Errorf("can't remove running container")
		return
	}

	// 删除容器工作空间
	err = DeleteWorkSpace(containerName, info.Volume)
	if err != nil {
		logrus.Errorf("delete work space, err: %v", err)
	}
	// 删除容器信息
	DeleteContainerInfo(containerName)
}
