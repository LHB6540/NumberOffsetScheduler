### Description
This is a custom scheduler for k8s. Its purpose is to allow statefulset pods to be deployed on optional nodes in a rolling manner according to their numbers, and to avoid pods with consecutive numbers from appearing on the same node. This is very useful for scenarios where business loads gradually reach pods according to pod numbers, such as game services corresponding to pod numbers.

### Deploy
The corresponding version products have been published to Docker Hub. You only need to execute the following command in the cluster to complete the deployment.
```
kubectl apply -f deploy/number-offset-scheduler.yaml
```
You can test by:
```
kubectl apply -f deploy/test.yaml
```
If you are using Open Kruise/Open Kruise Game, you can test by:
```
kubectl apply -f deploy/test-gss.yaml
```


### Development
The core logic is the scoring logic in pkg/scheduler.go. If you need to further modify the existing logic, you can execute
```
git clone <repo>
# add your code
go mod tidy
docker build -t number-offset-scheduler:latest  -f Dockerfile .
```
Done:
- fix: Function getMaxDiff just use filtered nodes, but not all nodes.Now is all nodes,but it works well.
- fix: log level;log detail;log format.

Todo:

### How it Work
This page will show you how it work and how to use it: 
[Configure Multiple Schedulers](https://kubernetes.io/docs/tasks/extend-kubernetes/configure-multiple-schedulers/)

Core score function:  
Example:   
There is Node 1、2、3,without any pods. 
```  
n1: []   
n2: []   
n3: []
````   
1、when pod-0 be scheduled,every node get 100 score,and pod will be bound to any node.   
2、If pod-0 had been bound to n2, when pod-1 be scheduled, n2 get score 1=2-1, n1 and n3 get score 100, and pod will be bound to n1 or n3.   
3、If pod-1 has been bound to n3, pod-2 will be bound to n1.   
4、Now，the node and pod Look like   
```
n1: pod-2   
n2: pod-0    
n3: pod-1   
```
5、When pod-3 be scheduled, n1 get score 1=3-2, n2 get score 1=3-0, n3 get score 2=3-1，and pod-3 will be bound to n2.   
6、If pod-0～pod-5 has been scheduled, and the nodes looks like this:  
```
n1: pod-2, pod-5  
n2: pod-0, pod-3  
n3: pod-1, pod-4  
```
when you delete pod-2 or recreate pod-0,the score will looks like: 
``` 
n1: 5(when no pod number is less than 2, get the min number as score)  
n2: 2=2-0(get the max number less than 2, get the diff as score)  
n3: 1=2-1(get the max number less than 2, get the diff as score)  
```
and pod-2 will be bound to n1.  
7、Some special cases: If you use some frameworks, such as Open Kruise/Open Kruise Game, the pod numbers may not be consecutive, and you will encounter situations like this:  
```
n1: pod-2, pod-105  
n2: pod-4, pod-107  
n3: pod-7, pod-108  
```
If you delete or recreate pod-4, the score will looks like: 
``` 
n1: 2=4-2(get the max number less than 4, get the diff as score)  
n2: 107(when no pod number is less than 4 get the min number as score)  
n3: 7(when no pod number is less than 4, get the min number as score)
```  
then the score of n1、n2 and n3 will be normailized to less than 100, and the score will be:  
```
n1: 1 = 2/2  
n2: 53 = 107/2  
n3: 3 = 7/2  
```

### Compatibility
[Scheduler Configuration](https://kubernetes.io/docs/reference/scheduling/config/) is now stable in version 1.25.
Tests have been run on versions 1.25 and 1.31.

### Reference Documents
- https://juejin.cn/post/7427399875236528191#heading-27
- https://github.com/cnych/sample-scheduler-framework/blob/master/main.go
- https://blog.csdn.net/weixin_43845924/article/details/138451208
- https://medium.com/@juliorenner123/k8s-creating-a-kube-scheduler-plugin-8a826c486a1
- https://overcast.blog/creating-a-custom-scheduler-in-kubernetes-a-practical-guide-2d9f9254f3b5
- https://arthurchiao.art/blog/k8s-scheduling-plugins-zh/
