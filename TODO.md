### Pending

- check for leaks - no nemory release after deleting workers, released with some delay (deferred GC? try runtime.GC())
- poor performace - 10 workers consume 4mb ram and 4% cpu, reuse Pinger instance?
- find better way to silence certain errors in "func (l WorkerLogger) Fatalf()"
- add some basic telemetry and configure graphana
- find root cause of the unreasonable docker image size growth from 7.7mb to 8.4mb
- check why device-pinger is reported by htop several times
- check why IPHONE_15_PRO_IP: 21:30:52 is OFFLINE 21:30:52 is ONLINE are received at the same time

### Completed

- (+) fix "go: warning: ignoring go.mod in $GOPATH /go" during docker build - reason is GOPATH and WORKDIR were pointing to same directory, and go do not expect to meet $GOPATH/go.mod (actually, according to commit, untill recently there was a fatal error instead of warning) https://groups.google.com/g/golang-codereviews/c/CmfKD9BXM2k and https://stackoverflow.com/questions/68232734/how-do-i-fix-gopath-go-mod-exists-but-should-not-linux-fedora
- (+) add linter - golangci-lint
- (+) add logging with level - switched to log/slog package
- (+) no gracefull shutdown inside docker
- (+) integrate into main docker-compose stack
- (+) send feedback via mqtt for add/del operations
- (+) rename mqttclient to mqtt
- (+) exclude "go test" from docker-build (to avoid installing go on build host)
- (+) push changes from MacMini
- (+) skip repeating errors like "no route to host"
- (+) implement mqtt "get" action
- (+) implement periodic status updates
- (+) fixed build commad for "exec format error"
- (+) split into worker and workers
- (+) workers.lock does not utilize full power of RWMutex
- (+) application is terminated if no target IP is set or all were deleted, to be fixed.
- (+) switch to https://github.com/sethvargo/go-envconfig
- (+) send first status update right after application startup