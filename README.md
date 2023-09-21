## Proxy Server

This repository contains a simple Go code for a reverse proxy server. The proxy server can be used to forward requests to one or more backend servers.

**Usage**

To run the proxy server, simply execute the following command:

"
go run main.go
"

The proxy server will start listening on port 8080 by default. You can access the proxy server at the following URL:

"
http://localhost:8080
"

**Configuration**

The proxy server can be configured using the following environment variables:

* `BACKEND_HOST`: The URL of the backend server to proxy.
* `DIRECTOR`: The name of a custom director function to use.

**Custom Director Functions**

A custom director function is a function that is responsible for choosing which backend server to forward a request to. To use a custom director function, you need to set the `DIRECTOR` environment variable to the name of the function.

Here is an example of a custom director function that proxies requests to two backend servers in a round-robin fashion:

"
func roundRobinDirector(req *http.Request) {
    // Get the list of backend servers.
    backendServers := []string{
        "http://localhost:8081",
        "http://localhost:8082",
    }

    // Choose a backend server in a round-robin fashion.
    nextBackend := backendServers[len(backendServers)-1]
    backendServers = append([]string{nextBackend}, backendServers[:len(backendServers)-1]...)

    // Set the target URL of the reverse proxy.
    req.URL.Host = nextBackend
}
"

To use this director function, you would set the `DIRECTOR` environment variable to `roundRobinDirector`.

**Modifying Requests and Responses**

The proxy server can also be used to modify requests and responses before they are forwarded to the backend server. To do this, you can use the `ModifyRequest()` and `ModifyResponse()` functions on the reverse proxy object.

For example, you could use the `ModifyRequest()` function to add a custom header to all requests:

"
reverseProxy := httputil.NewSingleHostReverseProxy(nil)
reverseProxy.Director = roundRobinDirector
reverseProxy.ModifyRequest = func(req *http.Request) {
    req.Header.Set("X-Custom-Header", "my-value")
}
"

Now, all requests made to the proxy server will have the `X-Custom-Header` header set to `my-value`.

You can also use the `ModifyResponse()` function to modify the response before it is sent to the client. For example, you could use it to remove a sensitive header from the response:

"
reverseProxy := httputil.NewSingleHostReverseProxy(nil)
reverseProxy.Director = roundRobinDirector
reverseProxy.ModifyResponse = func(res *http.Response) {
    res.Header.Del("X-Sensitive-Header")
}
"

Now, all responses sent by the proxy server will have the `X-Sensitive-Header` header removed.

**Conclusion**

The proxy server is a powerful tool that can be used to improve the performance, reliability, and security of your applications.
