# Tunneling tcp to tls written by Golang

## Install 

`go get github/liudanking/tcp2tls`

## Run 

`tcp2tls -l :21126 -r DST_TLS_ADDRESS -s true`

Parameter notes:

```
  -l string
    	local listen address (default "127.0.0.1:21126")
  -r string
    	remote TLS address
  -s	strict secure do NOT skip insecure certificate verify (default true)
```