# clusterclaims-controller

## Summary
When using Cluster pools, if you load this controller, it will create ManagedCluster, so that all you have to do is apply a simple `clusterClaim` resource.

```yaml
---
apiVersion: hive.openshift.io/v1
kind: ClusterClaim
metadata:
  name: my-cluster
  namespace: aws-east
spec:
  clusterPoolName: aws-eas
```
_Note: See `./examples/clusterclaim.yaml`_

## Deploy
To install this controller, you must be kubeadmin. The controller will be deployed to the `open-cluster-management` namespace.

```bash
oc apply -k ./deploy
```
It will take 1-2min for the image to download the first time. The controller runs two pods, and chooses a leader to reduce the possibility of an outage.

## Using ClusterClaims in GitOps

The following steps assume you will use the `./examples/clusterclaim.yaml`
1. Navigate to the ACM console and click `Infrastructure` > `Clusters` > `Cluster pools`
2. Click the `Create cluster pool` button
3. Choose `aws`, then pick an `Infrastructure provider credential`, and press `Next` to continue
4. Name the pool `aws-east` and use namespace `aws-east`, choose a size and release image, then click `Next` to continue
5. Customize any additional settings in steps 3-6
6. Then on step `7 Review` and click `Create` (To use Single node clusters, choose a 4.8.0 image and make sure the install-config has `controlPlane.replicas: 1` and `compute[0].replicas: 0`)
7. At this point you can apply the clusterclaim.yaml and a cluster will be claimed.

You can claim as many clusters as you want and they will be queued up and provisioned if the pool is empty.  It is recommended to create a subscription that points to the exmples folder, so you can commit more clusterclaim.yaml files to Git.  ACM will automatically claim those clusters, giving you a very simple Cluter Create GitOps flow.  You can create multiple pools, to support different cluster configurations and providers(AWS, GCP & Azure).

## Notes
* OpenShift GitOps can also be used to deliver the clusterclaim.yaml from the examples directory to the ACM Hub.
* When creating a namespace to hold your cluster pools, if you add the label:
  ```yaml
  metadata:
    labels:
      open-cluster-management.io/managed-by: clusterpools
      ...
  ```
  Then as the last cluster pool is removed, the namespace will be deleted. If the label is not present, the namespace will not be removed.
  
