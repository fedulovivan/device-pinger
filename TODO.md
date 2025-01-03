### Pending Prio 0

- check mhx19-next TODOs - store device names and mappings in db
- poor performace - 10 workers consume 4mb ram and 4% cpu, try Pinger instance polling?
- no retries after "Failed to complete pinger.Run()" worker is already marked as invalid and wont notice if device will return back online
- frequent "ERROR err="not Connected"" right after compose stack up
- for the http://macmini:8888/last-device-messages/192.168.88.44 align timestamp in "message.lastSeen" to match "timestamp"
- no new mqtt messages after mqtt disconnect/autoreconnect (`Connection lost error="pingresp not received, disconnecting"` and later `Connected broker=tcp://macmini:1883`) + same issue for device-pinger which impacts its service
- bug: some weird behavior after 5d uptime, no updates are sent, however mqtt api is alive (del/add/get-stats are working) - need doublecheck, looks everything is ok, kinda reproduced on 22 Oct after 1 month of uptime - do not observe feedback on any api call
  
### Pending Prio 1

- finish implementation for STATUS_INVALID
- find root cause of the unreasonable docker image size growth from 7.7mb to 8.4mb
- try error group instead of channels to collect workers errors

### Completed

- (+) check why device-pinger is reported by htop several times - not reproducable
- (+) add some basic telemetry and configure graphana
- (+) bug: check high goroutines count http://localhost:2112/debug/pprof/goroutine?debug=1 - with no workers 10 consumed by paho, others are - root, main x 2, RecordStartTime, os.signal, pprof, net.http
- (+) try: running with race flag
- (+) bug: goroutines leakage issue + https://github.com/golang/go/issues/2650 - goroutine with "for range" over ticker channel is blocked forever, since Ticker.C does not closed by design. solution: use "for + select + worker.done chan"
- (+) check for leaks - no nemory release after deleting workers, released with some delay (deferred GC? try runtime.GC())
- (+) learn which approach to use when we need to create several instances of pinger and spread "load" - introduce common storage with all ips with need to ping. eg redis. each app lock the ips pool, marks its slice, releases the lock
- (+) remove timestamps from "production" logger
- (+) handle "TZ=Europe/Moscow" in container
- (+) add git revision to the build
- (+) learn buffered channels - when you dont need to use channel for sync, just for async data transfer
- (+) non unique items TARGET_IPS make application to deadlock on shutdown - collection mutex issue
- (+) add errors to workers Add, Get, Delete, Create
- (+) use context to handle workers and application shutdown - tried, it brings more problems
- (+) rewrite packages to use init()
- (+) try reflection to parse Config struct tags and report unexpected variables from .env file
- (+) find better way to silence certain errors in "func (l WorkerLogger) Fatalf()"
- (+) send lastSeen in json
- (+) check why IPHONE_15_PRO_IP: 21:30:52 is OFFLINE 21:30:52 is ONLINE are received at the same time - presumably because of comparing lastSeen without checking for IsNull. status should not be set to OFFLINE untill first update for lastSeen will come
- (+) slog enable coloured output for development
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


### Code Trashbin (AKA It Has Potential, I Cannot Just Delete It!)

```golang

    // create root context to be cancelled with application shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// handle program shutdown
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt, syscall.SIGTERM,
	)
	defer stop()

	go func() {
		// nolint
		for {
			select {
			case <-ctx.Done():
				slog.Info("app termination signal received")
				workers.Wgg.Done()
				workers.StopAll()
				if workers.Errors != nil {
					close(workers.Errors)
				}
				return
			}
		}
	}()

    // declare custom primitive type with method which updates value by reference
    type MyInt int
    func (v *MyInt) Double() {
        *v = *v * 2
    }
	var mv MyInt = 2
	fmt.Println(mv)
	mv.Double()
	fmt.Println(mv)

    // task from https://go.dev/tour/methods/18
	type IPAddr [4]byte
	func (ip IPAddr) String() string {
		return strconv.Itoa(int(ip[0])) + "." + strconv.Itoa(int(ip[1])) + "." + strconv.Itoa(int(ip[2])) + "." + strconv.Itoa(int(ip[3]))
	}
    hosts := map[string]IPAddr{
        "loopback":  {127, 0, 0, 1},
        "googleDNS": {8, 8, 8, 8},
    }
    for name, ip := range hosts {
        fmt.Printf("%v: %v\n", name, ip)
    }

    // reading https://jordanorelli.com/post/32665860244/how-to-use-interfaces-in-go
    type Summer interface {
        Sum() int
        Add(v Summer) int
        Get() int
    }
    type impl struct {
        value int
    }
    func (this *impl) Sum() int {
        return this.value * 2
    }
    func (this *impl) Add(next Summer) int {
        return this.Get() + next.Get()
    }
    func (this *impl) Get() int {
        return this.value
    }
    func consumeSummer(s Summer) int {
        return s.Sum()
    }
    var i Summer = &impl{111}
    var k Summer = nil
    fmt.Printf("%T, %v\n", i, i)
    fmt.Printf("%T, %v\n", k, k)
    fmt.Printf("%v\n", k == nil)
    var j Summer = &impl{222}
    fmt.Printf("%v\n", i)
    fmt.Printf("%v\n", i.Sum())
    fmt.Printf("%T\n", i)
    fmt.Printf("%v\n", consumeSummer(i))
    fmt.Printf("%v\n", i.Add(j))


```