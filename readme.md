```shell
 for i in {0..10000}; do curl -X POST -H "Content-Type: application/json" -d '{"name": "idefav"}' http://192.168.0.105:15006/api/hello/ ;echo "" ; done
```