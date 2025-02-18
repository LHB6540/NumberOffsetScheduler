package scheduler

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	PluginName          = "number-offset-plugin"
	AnnotationKey       = "number-offset-plugin/scheduler-selector"
	//PodNumberAnnotation = "custom/pod-number"
)

type CustomScheduler struct {
	handle framework.Handle
}

var _ = framework.ScorePlugin(&CustomScheduler{})

// func New(_ runtime.Object, h framework.Handle) (framework.Plugin, error) {
// 	return &CustomScheduler{
// 		handle: h,
// 	}, nil
// }
func New(ctx context.Context, configuration runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	// 插件初始化逻辑
	return &CustomScheduler{handle: handle}, nil
}

func (p *CustomScheduler) Name() string {
	return PluginName
}

func (p *CustomScheduler) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := p.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return 0, framework.AsStatus(fmt.Errorf("getting node %q from Snapshot: %v", nodeName, err))
	}

	currentPodNumber, err := getPodNumber(pod)
	if err != nil {
		return 0, framework.AsStatus(fmt.Errorf("invalid annotation format for pod %s: %v", pod.Name, err))
	}

	selector := pod.Annotations[AnnotationKey]
	// AnnotationKey格式为k1=v1，如果是多个，使用","连接
	// 提取成selector列表，用于匹配，提取和匹配逻辑在matchesSelector函数中实现
	if selector == "" {
		return 0, framework.AsStatus(fmt.Errorf("annotation %s not found", AnnotationKey))
	}

	var nodePodNumbers []int
	for _, podInfo := range nodeInfo.Pods {
		p := podInfo.Pod
		if matchesSelector(p, selector) {
			nodePodNumber, err := getPodNumber(p)
			if err != nil {
				continue
			}
			nodePodNumbers = append(nodePodNumbers, nodePodNumber)
		}
	}

	// 打分逻辑
	// pod上面没有符合条件的pod，得分为100
	// pod上面有符合条件的pod，并根据上面的规则获取最大编号，
	// 1、如果大于当前待调度的pod，则向下取小于待调度的pod的最大值，如果一直向下取到没有则视为没有pod，直接打分100
	// 2、pod上面有符合条件的pod，并根据上面的规则获取最大编号，计算差值。同时获取所有节点上的最大编号的差值的最大值，以此为分母，当前节点差值为分子。乘以50进行归一化处理。
	var diff int
	if len(nodePodNumbers) == 0 {
		// 记录日志
		glog.Infof("nodeName: %s is empty, nodePodNumbers: %v", nodeName, nodePodNumbers)
		// 记录日志由于为空，打分100
		return 100, nil
	} else {
		sort.Ints(nodePodNumbers)
		maxLessThanCurrent := -1
		for _, number := range nodePodNumbers {
			if number < currentPodNumber && number > maxLessThanCurrent {
				maxLessThanCurrent = number
				//break
			}
		}
		//如果当前节点没有符合条件的pod，打分为51
		if maxLessThanCurrent == -1 {
			glog.Infof("nodeName: %s is not empty, but min nodePodNumbers: %v still greater than currentNumber ", nodeName, nodePodNumbers)
			return 51, nil
		} else {
			// 如果有符合条件的pod，调用函数获取所有节点上的最大编号的差值的最大值，以此为分母，当前节点差值为分子。乘以50进行归一化处理。
			diff = currentPodNumber - maxLessThanCurrent
			allMaxDiff := p.getMaxDiff(pod)
			// 记录节点信息
			glog.Infof("nodeName: %s, nodePodNumbers: %v", nodeName, nodePodNumbers)
			glog.Infof("maxDiff: %d, currentPodNumber: %d, maxLessThanCurrent: %d", allMaxDiff, currentPodNumber, maxLessThanCurrent)
			return int64(diff / allMaxDiff * 50), nil
		}
	}

}

func (p *CustomScheduler) ScoreExtensions() framework.ScoreExtensions {
	return p
}


// NormalizeScore implements framework.ScoreExtensions
func (*CustomScheduler) NormalizeScore(ctx context.Context, state *framework.CycleState, p *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	return nil
}


func (p *CustomScheduler) getMaxDiff(pod *v1.Pod) int {
	maxDiff := 0
	currentPodNumber, _ := getPodNumber(pod)
	selector := pod.Annotations[AnnotationKey]

	nodes, err := p.handle.SnapshotSharedLister().NodeInfos().List()
	if err != nil {
		glog.Errorf("Failed to list nodes: %v", err)
		return 100 // 默认值
	}

	for _, nodeInfo := range nodes {
		var nodePodNumbers []int
		for _, podInfo := range nodeInfo.Pods {
			p := podInfo.Pod
			if matchesSelector(p, selector) {
				nodePodNumber, _ := getPodNumber(p)
				nodePodNumbers = append(nodePodNumbers, nodePodNumber)
			}
		}

		if len(nodePodNumbers) == 0 {
			log.Printf("nodeName: %s is empty, don't change maxdiff ", nodeInfo.Node().Name, )
			continue
		}

		sort.Ints(nodePodNumbers)
		maxLessThanCurrent := -1
		for _, number := range nodePodNumbers {
			if number < currentPodNumber && number > maxLessThanCurrent {
				maxLessThanCurrent = number
			}
		}

		if maxLessThanCurrent != -1 {
			diff := currentPodNumber - maxLessThanCurrent
			if diff > maxDiff {
				maxDiff = diff
			}
		}
	}

	glog.Infof("maxDiff: %d", maxDiff)
	return maxDiff
}

func getPodNumber(pod *v1.Pod) (int, error) {
	// 从pod-name中提取pod的序号
	if pod.Name == "" {
		return 0, fmt.Errorf("pod name is empty")
	}
	// 使用分割符号提取pod的序号
	parts := strings.Split(pod.Name, "-")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid pod name format")
	}
	numberStr := parts[len(parts)-1]
	
	if numberStr == "" {
		return 0, fmt.Errorf("pod num is empty")
	}
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return 0, fmt.Errorf("annotation value is not a number: %s", numberStr)
	}
	return number, nil
}

func matchesSelector(pod *v1.Pod, selector string) bool {
	podLabels := pod.Labels
	selectorParts := strings.Split(selector, ",")
	for _, part := range selectorParts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			continue
		}
		key, value := kv[0], kv[1]
		// 判断key是否存在以及值是否相等
		if _, ok := podLabels[key]; !ok {
			return false
		}
		if podLabels[key] != value {
			return false
		}
	}
	return true
}


