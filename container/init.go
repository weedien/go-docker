package container

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
)

// 本容器执行的第一个进程
// 使用mount挂载proc文件系统
// 以便后面通过`ps`等系统命令查看当前进程资源的情况
func RunContainerInitProcess() error {
	cmdArray := readUserCommand()

	if len(cmdArray) == 0 {
		return fmt.Errorf("get user command in run container")
	}
	// 挂载
	err := setUpMount()
	if err != nil {
		logrus.Errorf("set up mount, err: %v", err)
		return err
	}

	// 获取当前的环境变量PATH的值
	currentPath := os.Getenv("PATH")

	// 检查是否已经包含/bin，如果不包含则在后面加上
	if !strings.Contains(currentPath, ":/bin") && !strings.Contains(currentPath, "=/bin") {
		newPath := currentPath + ":/bin"
		// 设置修改后的环境变量PATH
		os.Setenv("PATH", newPath)
	}

	// 在系统环境 PATH中寻找命令的绝对路径
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		path = cmdArray[0]
	}

	logrus.Infof("find cmd path: %v", path)

	err = syscall.Exec(path, cmdArray[0:], os.Environ())
	if err != nil {
		return err
	}

	return nil
}

// func receiveSignal() {
// 	// 创建一个信号通道用于接收信号
// 	sigCh := make(chan os.Signal, 1)

// 	// 将 SIGTERM 信号发送到 sigCh 通道
// 	signal.Notify(sigCh, syscall.SIGTERM)

// 	// 启动一个 goroutine 来处理信号
// 	go func() {
// 		// 阻塞直到收到信号
// 		<-sigCh

// 		// 删除容器工作空间
// 		err := DeleteWorkSpace(containerName, volume)
// 		if err != nil {
// 			logrus.Errorf("delete work space, err: %v", err)
// 		}
// 		// 删除容器信息
// 		DeleteContainerInfo(containerName)

// 		// 结束程序
// 		os.Exit(0)
// 	}()
// }

func readUserCommand() []string {
	// 指 index 为 3的文件描述符，
	// 也就是 cmd.ExtraFiles 中 我们传递过来的 readPipe
	pipe := os.NewFile(uintptr(3), "pipe")
	bs, err := ioutil.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("read pipe, err: %v", err)
		return nil
	}
	msg := string(bs)
	return strings.Split(msg, " ")
}

func setUpMount() error {
	err := pivotRoot()
	if err != nil {
		logrus.Errorf("pivot root, err: %v", err)
		return err
	}

	// systemd 加入linux之后, mount namespace 就变成 shared by default, 所以你必须显示
	//声明你要这个新的mount namespace独立。
	err = syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		return err
	}
	//mount proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	err = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		logrus.Errorf("mount proc, err: %v", err)
		return err
	}
	// mount temfs, temfs是一种基于内存的文件系统
	err = syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
	if err != nil {
		logrus.Errorf("mount tempfs, err: %v", err)
		return err
	}

	return nil
}

// 改变当前root的文件系统
func pivotRoot() error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	logrus.Infof("current location is %s", root)

	// systemd 加入linux之后, mount namespace 就变成 shared by default, 所以你必须显示
	//声明你要这个新的mount namespace独立。
	err = syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")
	if err != nil {
		return err
	}
	// 为了使当前root的老 root 和新 root 不在同一个文件系统下，我们把root重新mount了一次
	// bind mount是把相同的内容换了一个挂载点的挂载方法
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount rootfs to itself error: %v", err)
	}
	// 创建 rootfs/.pivot_root 存储 old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	_, err = os.Stat(pivotDir)
	if err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(pivotDir, 0777); err != nil {
			return err
		}
	}
	// pivot_root 到新的rootfs, 现在老的 old_root 是挂载在rootfs/.pivot_root
	// 挂载点现在依然可以在mount命令中看到
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivot_root %v", err)
	}
	// 修改当前的工作目录到根目录
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %v", err)
	}

	pivotDir = filepath.Join("/", ".pivot_root")
	// unmount rootfs/.pivot_root
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir %v", err)
	}
	// 删除临时文件夹
	return os.Remove(pivotDir)
}
