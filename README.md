# cma-operator
Cluster Manager API Operator

### Deployment
The default way to deploy CMA-Operator is by the provided helm chart located in the `deployment/helm/cma-operator` directory.

#### Prerequisites
1. [ingress controller](https://github.com/helm/charts/tree/master/stable/nginx-ingress)


Optional steps when using CMA API Proxy:
1. Create proxy DNS name pointing to nginx ingress controller load balancer name or IP, depending on your setup.
1. Update the following entries in the values.yaml
    * `cma.enabled: true`
    * `cma.apiProxyEndpoint: <dns name from step above>`
    * `cma.apiProxyTls: <name of tls secret` (if using secure) or  `cma.insecure: true` for insecure.
    
    All new managed clusters will be created with an [ingress](https://kubernetes.io/docs/concepts/services-networking/ingress) and [ExternalName](https://kubernetes.io/docs/concepts/services-networking/service/#externalname) service

#### install via [helm](https://helm.sh/docs/using_helm/#quickstart)
1. Install helm chart without default bundles first:
    ```bash
    helm install deployments/helm/cma-operator --name cma-operator \
        --set bundles.metrics=false \
        --set bundles.nginxk8smon=false \
        --set bundles.nodelabelbot5000=false
    ```
    *alternatively you can update `values.yaml`.
    
    After the `cma-operator` pods are running you can upgrade the helm release to include the bundles, see more on [SDSAppBundle](#How-to-create-a-SDSAppBundle).
    ```bash
    helm upgrade cma-operator deployments/helm/cma-operator \
        --set bundles.metrics=true \
        --set bundles.nginxk8smon=true \
        --set bundles.nodelabelbot5000=true
    ```

#### Using the CMA API Proxy to access clusters (via ingress proxy):
1. retrieve the `bearertoken` from a GetCluster() call at the CMA level: 
    * example output:
    ```
    {
         "ok": true,
         "cluster": {
           "id": "clusterExampleID",
           "name": "FooCluster1",
           "kubeconfig": "apiVersion: v1
            clusters:
            - cluster:
                certificate-authority-data: <REDACTED>
                server: https://foocluster1.cloud.com
              name: foocluster1
            contexts:
            - context:
                cluster: foocluster1
                user: clusterAdmin_foocluster1
              name: foocluster1
            current-context: foocluster1
            kind: Config
            preferences: {}
            users:
            - name: clusterAdmin_foocluster1
              user:
                client-certificate-data: <REDACTED>
                client-key-data: <REDACTED>,
           "status": "RUNNING",
           "bearertoken": "etJhbGciOiJSUzI1NiLsImtpZCI6IiJ9.eyJpc3MiOiJrdWJlcm5ldGVzL3NlcnZpY2VhY2NvdW50Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9uYW1lc3BhY2UiOiJkZWZhdWx0Iiwia3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudC9zZWNyZXQubmFtZSI6InNkcy1zYS10b2tlbi1zNzhiOCIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50Lm5hbWUiOiJzZHMtc2EiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC51aWQiOiJlNWQ5OTQ5OF0wZWRmLTExZTktOWNlYi01YTk5N2S2MGU0YWEiLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6ZGVmYXVsdDpzZHMtc2EifQ.RLkxZI3nPDvPzTzKBkxVR4cX5Jw0PgkYlm2z343SWxOD6Eylf16xHqUfZxaJ5jVPXT80q5alKtjfR8OMNXC93YdmQrZsdRAFuOkwJ8u1Tk1_u7-njvkyKNemB7iqcJRlLPzzlVaR29kC6WpHEWmARZK2jpu7Q7RVc8GNozpmmeCUcHmzllLz2ueoSDdp5pGf2zOpNxOgU_r4eCcj1VuL-i8SVIjfs2iosO1GgP4KU1nwzZESruDnXHw1ZskjBwBb2nordHcYTwzva1S5VRcaGzmv6SqUXyWeuNeZAr-SfLgxF3pU0KXYLUOvSYZQBwvIblu9CubihUojsO9N7_rn4BHAkn_oNADppCvgdQaIjYH35fAW6_86NumD-zLneJ5as32X5IiulwIIqmq4tfo9KkupN3-yMgvPYdzcZq6E5V7l1WbHEY7-uOqkKFcvShZ_0KY7prjgp5BQR3C45IoyIOvd_iKYiPECUum96_RTangKXVK7m77znkGa_zbbVlfyNoDacKtV0TxikRfiv2LrZxfMbp3TsS4vD4-xAWVRoaqvNqJplHQYmVr45BWeVxTnelszvOK1tTvA4r30y9_0lW1CQz1Pj9x9HmNiRZ02ot-OeRHaP367wmh0sr2Kj15JHOMCrTOBT2-5kY-Wf_xlwJ5hli_RWmtFG9EddGa7JTQ"
         }
       }
    ```

2. Using the "bearertoken" from the output above you can now make api calls to the managed cluster:
    * example:
    `curl https://<ProxyDNSname>/<clusterName>/api/v1/<API path> -H "Authorization: Bearer <bearerToken>" [--insecure]`

#### Custom Resources
* SDSClusters
* SDSPackageManagers
* SDSApplications
* SDSAppBundles


#### How to create new Custom Resources in this repo:


#### How to test this repo:

**Manually**:
1. run cluster-manager-api (see repo for details)
2. confirm your `kubectl config current-context` is pointing to the cluster you want to test with (ex: minikube)
3. run the following command from the root of the cma-operator directory 
```
CMAOPERATOR_CMA_ENDPOINT=localhost:9050 CMAOPERATOR_CMA_INSECURE=true go run cmd/cma-operator/main.go
```
* note the environment variables are pointing to the location of the cluster-manager-api from step 1. 

Additional flags available: (add to the end of command above)
* proxy test: `--cma-api-proxy sample.example.com`

**Using Opctl**:

requirements:
1. docker
2. [opctl](https://opctl.io/docs/getting-started/opctl.html#installation)

steps:
1. start cluster-manager-api using opctl (see repo for details)
2. start debug op: `opctl run debug`

This will start a container running `cma-operator` and referencing a [kind](https://github.com/kubernetes-sigs/kind) cluster created by the cluster-manager-api debug op

### How to create a SDSAppBundle:

SDSAppBundles are a way to define what helm charts get installed on managed clusters automatically on cluster creation.

The current parameter to define what type of cluster to install on is based on the "Provider". 
This can be either one or more provider or **empty** for all providers.

`providers: [ aws, azure, vmware, ssh ]`

example yaml for creating a bundle:

```yaml
   apiVersion: cma.sds.samsung.com/v1alpha1
   kind: SDSAppBundle
   metadata:
     name: ingress-controller-bundle
     namespace: default
   spec:
     applications:
     - chart:
         chartName: nginx-ingress
         repository:
           name: charts
           url: https://github.com/helm/charts/stable
         version: 1.1.4
       name: sample-ingress-controller
       namespace: default
       packageManager:
         name: ingress-tiller
       values: |
        controller.metrics.enabled=true,
        controller.stats.enabled=true,
     k8sversion: 1.10.6
     name: ingress-controller-bundle
     namespace: default
     packagemanager:
       image: gcr.io/kubernetes-helm/tiller
       name: ingress-tiller
       namespace: default
       permissions:
         clusterWide: false
         namespaces:
         - default
       serviceAccount:
         name: ingress-sa
         namespace: default
       version: v2.11.0
     providers: [ 'aws', 'azure' ]
```

