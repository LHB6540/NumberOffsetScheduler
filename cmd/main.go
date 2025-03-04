package main

import (
	"fmt"
	plugin "number-offset-scheduler/pkg"
	"k8s.io/component-base/logs"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
	"os"
)

func main() {
	command := app.NewSchedulerCommand(
		app.WithPlugin(plugin.PluginName, plugin.New),
	)

	logs.InitLogs()
	// 打印到控制台
	// logs.AddFlags(nil)
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

}