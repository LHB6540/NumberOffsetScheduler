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
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

const (
	PluginName          = "number-offset-scheduler"
	AnnotationKey       = "number-offset-scheduler/scheduler-selector"
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
	log.Printf("!!!!!!!!!Now Scoring NodeName: %s",nodeName)
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
	// 情况1:节点上面没有符合条件的pod，得分为0，这是一种特殊情况。最终在归一化函数中得分将会最大。
	// 情况2：节点上面有符合条件的pod，并且存在小于待调度pod的变好的pod，取节点上小于当前pod的pod的最大值，使用当前pod的编号减去最大小于当前pod的编号，作为得分，如果最终有大于100的，在归一化函数中处理。
	// 情况3: 节点上面有符合条件的pod，但是最小的pod大于待调度pod，取节点上最小的pod编号作为得分。如果最终结果大于100，在归一化函数中处理。
	
	// 旧逻辑：
	// 1、如果大于当前待调度的pod，则向下取小于待调度的pod的最大值，如果一直向下取到没有则视为没有pod，直接打分100
	// 2、pod上面有符合条件的pod，并根据上面的规则获取最大编号，计算差值。同时获取所有节点上的最大编号的差值的最大值，以此为分母，当前节点差值为分子。乘以50进行归一化处理。
	var diff int
	if len(nodePodNumbers) == 0 {
		// 记录日志
		glog.Infof("nodeName: %s is empty, nodePodNumbers: %v", nodeName, nodePodNumbers)
		// 记录日志由于为空，打分100
		return 0, nil
	} else {
		sort.Ints(nodePodNumbers)
		maxLessThanCurrent := -1
		for _, number := range nodePodNumbers {
			if number < currentPodNumber && number > maxLessThanCurrent {
				maxLessThanCurrent = number
				//break
			}
		}
		//如果当前节点没有符合条件的pod，取当前pod的编号作为得分
		if maxLessThanCurrent == -1 {
			glog.Infof("nodeName: %s is not empty, but min nodePodNumbers: %v still greater than currentNumber ", nodeName, nodePodNumbers)
			// 返回pods中的最小值
			return int64(nodePodNumbers[0]), nil
		} else {
			// 如果有符合条件的pod，取差值
			diff = currentPodNumber - maxLessThanCurrent
			return int64(diff), nil
			// allMaxDiff := p.getMaxDiff(pod)
			// // 记录节点信息
			// glog.Infof("nodeName: %s, nodePodNumbers: %v", nodeName, nodePodNumbers)
			// glog.Infof("maxDiff: %d, currentPodNumber: %d, maxLessThanCurrent: %d", allMaxDiff, currentPodNumber, maxLessThanCurrent)
			// return int64(diff / allMaxDiff * 50), nil
		}
	}

}

func (p *CustomScheduler) ScoreExtensions() framework.ScoreExtensions {
	return p
}


// NormalizeScore implements framework.ScoreExtensions
func (*CustomScheduler) NormalizeScore(ctx context.Context, state *framework.CycleState, p *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	log.Printf("!!!!!!!!!NormalizeScore is called, NodeScoreList: %v",scores)
	// 在此处对分数进行归一化处理，如果分数为0，最后变为最大值100。
	// 获取所有不为0的分数，取最大的分数
	// 逐渐测试一个压缩比例，使得最大的分数能压缩到0-100之间
	// 将这个比例应用到所有不为100的分数上
	maxDiff := int64(0)
	for i := range scores {
		if scores[i].Score != 0 {
			if scores[i].Score > maxDiff {
				maxDiff = scores[i].Score
			}
		}
	}
	if maxDiff == 0 {
		return nil
	}
	compress := int64(1)
	if maxDiff > 100 {
		compress := int64(1)
		for maxDiff > 100 {
			compress = compress * 2
			maxDiff = maxDiff / compress
		}
	}
	for i := range scores {
		if scores[i].Score == 0 {
			scores[i].Score = 100
		}else {
			scores[i].Score = int64(scores[i].Score / compress)
		}
	}
	log.Printf("!!!!!!!!!NormalizeScore is called, NodeScoreList: %v",scores)
	klog.Warningf("!!!!!!! Log by log, NodeScoreList: %v",scores)
	return nil
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


