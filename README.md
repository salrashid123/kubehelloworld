###Kubernetes Services HelloWorld

Sample application that deploys a trivial [Kubernetes service](https://github.com/kubernetes/kubernetes/blob/release-1.0/docs/user-guide/services.md) to connect a frontend system (fe) with a backend (be).  

This is merely to demonstrate Kubernetes service discovery in [Google Container Engine (GKE)](https://cloud.google.com/container-engine/docs/services/), nothing more and is based off the guestbook service example.

Parts of this sample code is from the default kubernetes 'environment-guide' example.  


HTTP requests to the frontend handler causes a service lookup for the backend and retrieves some data from the backend.  

`user` --> `GKE Ingress` --> `frontend` --> lookup backend service --> make backend API call --> `backend` --> return data to frontend --> return web page to user showing some data from the backend.  



### Create Test Cluster

Create the cluster with three nodes in us-central1-a using either gcloud or the Cloud Console

```bash
gcloud config set compute/zone us-central1-a
gcloud container  clusters create cluster-1  --zone us-central1-a  --num-nodes 3 --enable-ip-alias
```

### Create Deployments, services, Ingress

Run the following to create the frontend/backend replication controllers and services.


```bash
$ kubectl apply -f .

# wait about 8mins until an ingress IP has been assigned

$ kubectl get no,po,svc,ing
NAME                                            STATUS   ROLES    AGE   VERSION
node/gke-cluster-1-default-pool-9aa48162-7g9v   Ready    <none>   87m   v1.19.9-gke.1400
node/gke-cluster-1-default-pool-9aa48162-nc2x   Ready    <none>   87m   v1.19.9-gke.1400

NAME                                    READY   STATUS    RESTARTS   AGE
pod/be-deployment-7bdd897fb8-8xs2d      1/1     Running   0          17m
pod/be-deployment-7bdd897fb8-pqv2c      1/1     Running   0          17m
pod/myapp-deployment-5676b87c78-fnspb   1/1     Running   0          17m
pod/myapp-deployment-5676b87c78-rsjqc   1/1     Running   0          17m

NAME                 TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
service/be-srv       ClusterIP   10.10.40.24    <none>        5000/TCP   17m
service/kubernetes   ClusterIP   10.10.32.1     <none>        443/TCP    87m
service/myapp-srv    ClusterIP   10.10.36.226   <none>        8080/TCP   17m

NAME                                          CLASS    HOSTS   ADDRESS         PORTS     AGE
ingress.networking.k8s.io/myapp-srv-ingress   <none>   *       34.120.236.42   80, 443   17m


export INGRESS_IP=`kubectl -n default get ingress.networking.k8s.io/myapp-srv-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`
echo $INGRESS_IP
```




### Test the GKE cluster

The frontend service is available **34.120.236.42** so an invocation shows:

```bash
$ curl -vk https://$INGRESS_IP/

*   Trying 34.120.236.42:443...
* Connected to 34.120.236.42 (34.120.236.42) port 443 (#0)

* TLSv1.3 (OUT), TLS handshake, Client hello (1):
* TLSv1.3 (IN), TLS handshake, Server hello (2):
* TLSv1.3 (IN), TLS handshake, Encrypted Extensions (8):
* TLSv1.3 (IN), TLS handshake, Certificate (11):
* TLSv1.3 (IN), TLS handshake, CERT verify (15):
* TLSv1.3 (IN), TLS handshake, Finished (20):
* TLSv1.3 (OUT), TLS change cipher, Change cipher spec (1):
* TLSv1.3 (OUT), TLS handshake, Finished (20):
* SSL connection using TLSv1.3 / TLS_AES_256_GCM_SHA384
* ALPN, server accepted to use h2
* Server certificate:
*  subject: C=US; ST=California; O=Google; OU=Enterprise; CN=gke.esodemoapp2.com
*  start date: Mar 30 22:07:36 2016 GMT
*  expire date: Apr  9 22:07:36 2017 GMT
*  issuer: C=US; ST=California; L=Mountain View; O=Google; OU=Enterprise; CN=TestCAforESO
*  SSL certificate verify result: unable to get local issuer certificate (20), continuing anyway.


> GET / HTTP/2
> Host: 34.120.236.42
> user-agent: curl/7.74.0
> accept: */*
> 
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
* TLSv1.3 (IN), TLS handshake, Newsession Ticket (4):
* old SSL session ID is stale, removing
* Connection state changed (MAX_CONCURRENT_STREAMS == 100)!
< HTTP/2 200 
< date: Thu, 27 May 2021 22:57:57 GMT
< content-length: 620
< content-type: text/plain; charset=utf-8
< via: 1.1 google
< alt-svc: clear
< 


SRV CNAME: _be._tcp.be-srv.default.svc.cluster.local.
SRV Records: &{be-srv.default.svc.cluster.local. 5000 10 100} 
Pod Name:  
Pod Namespace:  
USER_VAR:  

Kubenertes environment variables
BE_SRV_SERVICE_HOST = 10.10.40.24 
BE_SRV_SERVICE_PORT = 5000 
KUBERNETES_SERVICE_HOST = 10.10.32.1 
KUBERNETES_SERVICE_PORT = 443 
MYAPP_SRV_SERVICE_HOST = 10.10.36.226 
MYAPP_SRV_SERVICE_PORT = 8080 

Found ENV lookup backend ip: 10.10.40.24 port: 5000
ENV Lookup Response from backend
BACKEND Response
Found DNS lookup backend ip: be-srv.default.svc.cluster.local. port: 5000
DNS Lookup Response from backend
```


The above output is from the frontend and shows the backend discovery by both environment variables and DNS SRV.   The response from the backend is just the part **BACKEND Response**.  

The output shows the frontend discovered the backend using envionment variables  IP address/port values:   **10.167.252.252:5000**  

The output also shows the DNS SRV request contained the host and port to connect to from the frontend:  **be-srv.default.svc.cluster.local. port: 5000**


### Frontend/backend Services

The frontend consists of a golang program that listens for http requests on port :8080.  It is wrapped in the container  

[salrashid123/fe](https://hub.docker.com/r/salrashid123/fe/)

The frontend service is configured to startup 2 replicas inside a Loaldbalanced Kubernetes service:

**Frontend**

##### Frontend Service Definition

Frontend is a simple go web application which makes an api call to the backend.  It returns the information about on the current container the cluster is in as well as the backend service's response

see code in `fe/fe.go` folder
#### Backend

The backend consists of another golang program which for http requests on port :5000. All that it does is echos back the container name

```golang
package main

import (
	"fmt"
	"log"
	"net/http"
)

func printInfo(resp http.ResponseWriter, req *http.Request) {

	fmt.Fprintf(resp, "BACKEND Response")

}

func main() {
	http.HandleFunc("/", printInfo)
	log.Fatal(http.ListenAndServe(":5000", nil))
}
```


### Discovery

Once a request hits any pod running the frontend service, the front end attempts to discover how to connect to the backend service.  This is done in two ways:  using environment variables or (preferably), by DNS SRV lookups.  
Each node in the cluster runs a local [DNS](https://github.com/kubernetes/kubernetes/tree/master/cluster/addons/dns) server.  

Also see [Kubernetes networking](https://github.com/kubernetes/kubernetes/blob/release-1.0/docs/design/networking.md)

**Environment Variables**
```go
    backendHost := os.Getenv("BE_SRV_SERVICE_HOST")
    backendPort := os.Getenv("BE_SRV_SERVICE_PORT")
    backendRsp, backendErr := http.Get(fmt.Sprintf(
        "http://%v:%v/",
        backendHost,
        backendPort))
    if backendErr == nil {
        defer backendRsp.Body.Close()
    }
```

**DNS SRV**
```go
    cname, rec, err := net.LookupSRV("be", "tcp", "be-srv.default.svc.cluster.local")
    if err != nil {
        http.Error(resp, err.Error(), http.StatusInternalServerError)
    }
    fmt.Fprintf(resp, "SRV CNAME: %v\n", cname)
    for i := range rec {
        fmt.Fprintf(resp, "SRV Records: %v \n", rec[i])
        DNSbackendHost = rec[i].Target
        DNSbackendPort = strconv.Itoa(int(rec[i].Port))
    }
```

