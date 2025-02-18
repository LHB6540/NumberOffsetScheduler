### Description
This is a custom scheduler for k8s. Its purpose is to allow statefulset pods to be deployed on optional nodes in a rolling manner according to their numbers, and to avoid pods with consecutive numbers from appearing on the same node. This is very useful for scenarios where business loads gradually reach pods according to pod numbers, such as game services corresponding to pod numbers.

### deploy
The corresponding version products have been published to Docker Hub. You only need to execute the following command in the cluster to complete the deployment.
```
kubectl apply -f deploy/number-offset-scheduler.yaml
```

### development
The core logic is the scoring logic in pkg/scheduler.go. If you need to further modify the existing logic, you can execute
```
git clone <repo>
# add your code
go mod tidy
docker build -t number-offset-scheduler:latest  -f Dockerfile .
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
