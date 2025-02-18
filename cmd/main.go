// package main

// import (
// 	//"flag"
// 	"fmt"
// 	"os"
// 	"time"

// 	plugin "numberoffsetscheduler/pkg"

// 	//"k8s.io/klog/v2"
// 	"golang.org/x/exp/rand"
// 	"k8s.io/kubernetes/cmd/kube-scheduler/app"
// )

// func main() {
//     rand.Seed(time.Now().UTC().UnixNano())

//     command := app.NewSchedulerCommand(
//         app.WithPlugin(sample.Name, sample.New), 
//     )

//     logs.InitLogs()
//     defer logs.FlushLogs()

//     if err := command.Execute(); err != nil {
//         _, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
//         os.Exit(1)
//     }

// }

// // func main() {
// // 	//flag.Parse()
// // 	// klog.InitFlags(nil)
// // 	// defer klog.Flush()

// // 	command := app.NewSchedulerCommand(
// // 		app.WithPlugin(plugin.PluginName, plugin.New),
// // 	)

// // 	if err := command.Execute(); err != nil {
// // 		//klog.Errorf("Error executing scheduler command: %v", err)
// // 		fmt.Fprint(os.Stderr,"%v\n",err)
// // 		os.Exit(1)
// // 		return
// // 	}
// // }



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
	logs.AddFlags(nil)
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

}