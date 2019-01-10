# cma-operator
Cluster Manager API Operator

### How to use CMA API Proxy

#### installation steps:
1. deploy [nginx ingress controller](https://github.com/helm/charts/tree/master/stable/nginx-ingress)
```bash
helm install stable/nginx-ingress
```
2. Create Proxy DNS name pointing to nginx ingress controller load balancer name or IP, depending on your setup.
3. Deploy helm chart for `cma-operator` updating the following entries in the values.yaml
* `cma.enabled: true`
* `cma.apiProxyEndpoint: <dns name from step 2 above>`
* `cma.apiProxyTls: <name of tls secret` (if using secure) or  `cma.insecure: true` for insecure.

All new managed clusters will be created with an [ingress](https://kubernetes.io/docs/concepts/services-networking/ingress) and [ExternalName](https://kubernetes.io/docs/concepts/services-networking/service/#externalname) service

#### how to access clusters via ingress:
1. retrieve the `bearertoken` from a GetCluster() call: 
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
1. run cluster-manager-api (see repo for details)
2. confirm your `kubectl config current-context` is pointing to the cluster you want to test with (ex: minikube)
3. run the following command from the root of the cma-operator directory 
```
CMAOPERATOR_CMA_ENDPOINT=localhost:9050 CMAOPERATOR_CMA_INSECURE=true go run cmd/cma-operator/main.go
```
* note the environment variables are pointing to the location of the cluster-manager-api from step 1. 

Additional flags available: (add to the end of command above)
* proxy test: `--cma-api-proxy sample.example.com`

