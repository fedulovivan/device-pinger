handle:
```txt
panic: write udp 0.0.0.0:0->192.168.88.62:0: sendto: no route to host

goroutine 50 [running]:
main.SpawnWorker.func3()
	/Users/ivanf/Desktop/Projects/go/device-pinger/main.go:135 +0x7c
created by main.SpawnWorker in goroutine 1
	/Users/ivanf/Desktop/Projects/go/device-pinger/main.go:132 +0x378
exit status 2
```