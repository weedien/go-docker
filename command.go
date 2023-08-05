package main

import (
	"fmt"
	"go-docker/common"
	"go-docker/network"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"go-docker/cgroups/subsystem"
	"go-docker/container"
)

// 创建namespace隔离的容器进程
// 启动容器
var runCommand = cli.Command{
	Name:  "run",
	Usage: "Create a container with namespace and cgroups limit",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "ti",
			Usage: "enable tty",
		},
		cli.StringFlag{
			Name:  "m",
			Usage: "memory limit",
		},
		cli.StringFlag{
			Name:  "cpushare",
			Usage: "cpushare limit",
		},
		cli.StringFlag{
			Name:  "cpuset",
			Usage: "cpuset limit",
		},
		cli.StringFlag{
			Name:  "v",
			Usage: "docker volume",
		},
		cli.BoolFlag{
			Name:  "d",
			Usage: "detach container",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "container name",
		},
		cli.StringSliceFlag{
			Name:  "e",
			Usage: "docker env",
		},
		cli.StringFlag{
			Name:  "net",
			Usage: "container network",
		},
		cli.StringSliceFlag{
			Name:  "p",
			Usage: "port mapping",
		},
	},
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container args")
		}

		res := &subsystem.ResourceConfig{
			MemoryLimit: ctx.String("m"),
			CpuSet:      ctx.String("cpuset"),
			CpuShare:    ctx.String("cpushare"),
		}
		// cmdArray 为容器运行后，执行的第一个命令信息
		// cmdArray[0] 为镜像名, .Tail() 是去掉第一个后的全部参数
		var cmdArray []string
		for _, arg := range ctx.Args().Tail() {
			cmdArray = append(cmdArray, arg)
		}

		tty := ctx.Bool("ti")
		detach := ctx.Bool("d")

		if tty && detach {
			return fmt.Errorf("ti and d paramter can not both provided")
		}

		containerName := ctx.String("name")
		volume := ctx.String("v")
		net := ctx.String("net")
		// 要运行的镜像名
		imageName := ctx.Args().Get(0)
		envs := ctx.StringSlice("e")
		ports := ctx.StringSlice("p")

		Run(cmdArray, tty, res, containerName, imageName, volume, net, envs, ports)
		return nil
	},
}

// 初始化容器内容,挂载proc文件系统，运行用户执行程序
var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	Action: func(ctx *cli.Context) error {
		logrus.Infof("init come on")
		return container.RunContainerInitProcess()
	},
}

// 导出容器内容
var commitCommand = cli.Command{
	Name:  "commit",
	Usage: "docker commit a container into image",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "c",
			Usage: "export image path",
		},
	},
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		imageName := ctx.Args().Get(0)
		imagePath := ctx.String("c")
		return container.CommitContainer(imageName, imagePath)
	},
}

var listCommand = cli.Command{
	Name:  "ps",
	Usage: "list all container",
	Action: func(ctx *cli.Context) error {
		container.ListContainerInfo()
		return nil
	},
}

var logCommand = cli.Command{
	Name:  "logs",
	Usage: "look container log",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing container name")
		}
		containerName := ctx.Args().Get(0)
		container.LookContainerLog(containerName)
		return nil
	},
}

var execCommand = cli.Command{
	Name:  "exec",
	Usage: "exec a command into container",
	Action: func(ctx *cli.Context) error {
		// 如果环境变量里面有 PID,那么则什么都不执行
		pid := os.Getenv(common.EnvExecPid)
		if pid != "" {
			logrus.Infof("pid callback pid %s, gid: %d", pid, os.Getgid())
			return nil
		}
		if len(ctx.Args()) < 2 {
			return fmt.Errorf("missing container name or command")
		}

		var cmdArray []string
		for _, arg := range ctx.Args().Tail() {
			cmdArray = append(cmdArray, arg)
		}

		containerName := ctx.Args().Get(0)
		container.ExecContainer(containerName, cmdArray)
		return nil
	},
}

var stopCommand = cli.Command{
	Name:  "stop",
	Usage: "stop a container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing stop container name")
		}
		containerName := ctx.Args().Get(0)
		container.StopContainer(containerName)
		return nil
	},
}

var removeCommand = cli.Command{
	Name:  "rm",
	Usage: "rm a container",
	Action: func(ctx *cli.Context) error {
		if len(ctx.Args()) < 1 {
			return fmt.Errorf("missing remove container name")
		}
		containerName := ctx.Args().Get(0)
		container.RemoveContainer(containerName)
		return nil
	},
}

var networkCommand = cli.Command{
	Name:  "network",
	Usage: "container network commands",
	Subcommands: []cli.Command{
		{
			Name:  "create",
			Usage: "create a container network",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "driver",
					Usage: "network driver",
				},
				cli.StringFlag{
					Name:  "subnet",
					Usage: "subnet cidr",
				},
			},
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("Missing network name")
				}
				err := network.Init()
				if err != nil {
					logrus.Errorf("network init failed, err: %v", err)
					return err
				}
				// 创建网络
				err = network.CreateNetwork(context.String("driver"), context.String("subnet"), context.Args()[0])
				if err != nil {
					return fmt.Errorf("create network error: %+v", err)
				}
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "list container network",
			Action: func(context *cli.Context) error {
				err := network.Init()
				if err != nil {
					logrus.Errorf("network init failed, err: %v", err)
					return err
				}

				network.ListNetwork()
				return nil
			},
		},
		{
			Name:  "remove",
			Usage: "remove container network",
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("Missing network name")
				}

				err := network.Init()
				if err != nil {
					logrus.Errorf("network init failed, err: %v", err)
					return err
				}
				err = network.DeleteNetwork(context.Args()[0])
				if err != nil {
					return fmt.Errorf("remove network error: %+v", err)
				}
				return nil
			},
		},
	},
}
