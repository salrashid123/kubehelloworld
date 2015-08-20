package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

func getKubeEnv() (map[string]string, error) {
	environS := os.Environ()
	environ := make(map[string]string)
	for _, val := range environS {
		split := strings.SplitN(val, "=", 2)
		if len(split) != 2 {
			return environ, fmt.Errorf("Some weird env vars")
		}
		environ[split[0]] = split[1]
	}
	for key := range environ {
		if !(strings.HasSuffix(key, "_SERVICE_HOST") ||
			strings.HasSuffix(key, "_SERVICE_PORT")) {
			delete(environ, key)
		}
	}
	return environ, nil
}

func printInfo(resp http.ResponseWriter, req *http.Request) {
	kubeVars, err := getKubeEnv()
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	backendHost := os.Getenv("BE_SRV_SERVICE_HOST")
	backendPort := os.Getenv("BE_SRV_SERVICE_PORT")
	backendRsp, backendErr := http.Get(fmt.Sprintf(
		"http://%v:%v/",
		backendHost,
		backendPort))
	if backendErr == nil {
		defer backendRsp.Body.Close()
	}

	/*
		addr, err := net.LookupHost("be-srv")
		if err != nil {
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
	*/

	DNSbackendHost := ""
	DNSbackendPort := ""
	cname, rec, err := net.LookupSRV("be", "tcp", "be-srv.default.svc.cluster.local")
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintf(resp, "SRV CNAME: %v\n", cname)
	//iterate through the records and pick one
	for i := range rec {

		fmt.Fprintf(resp, "SRV Records: %v \n", rec[i])
		DNSbackendHost = rec[i].Target
		DNSbackendPort = strconv.Itoa(int(rec[i].Port))
	}

	DNSbackendRsp, DNSbackendErr := http.Get(fmt.Sprintf(
		"http://%v:%v/",
		DNSbackendHost,
		DNSbackendPort))
	if DNSbackendErr == nil {
		defer DNSbackendRsp.Body.Close()
	}

	name := os.Getenv("POD_NAME")
	namespace := os.Getenv("POD_NAMESPACE")
	fmt.Fprintf(resp, "Pod Name: %v \n", name)
	fmt.Fprintf(resp, "Pod Namespace: %v \n", namespace)

	envvar := os.Getenv("USER_VAR")
	fmt.Fprintf(resp, "USER_VAR: %v \n", envvar)

	fmt.Fprintf(resp, "\nKubenertes environment variables\n")
	var keys []string
	for key := range kubeVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(resp, "%v = %v \n", key, kubeVars[key])
	}

	fmt.Fprintf(resp, "\nFound ENV lookup backend ip: %v port: %v\n", backendHost, backendPort)
	if backendErr == nil {
		fmt.Fprintf(resp, "ENV Lookup Response from backend\n")
		io.Copy(resp, backendRsp.Body)
	} else {
		fmt.Fprintf(resp, "Error from backend: %v", backendErr.Error())
	}

	fmt.Fprintf(resp, "\nFound DNS lookup backend ip: %v port: %v\n", DNSbackendHost, DNSbackendPort)
	if DNSbackendErr == nil {
		fmt.Fprintf(resp, "DNS Lookup Response from backend\n")
		io.Copy(resp, DNSbackendRsp.Body)
	} else {
		fmt.Fprintf(resp, "Error from backend: %v", DNSbackendErr.Error())
	}

}

func main() {
	http.HandleFunc("/", printInfo)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
