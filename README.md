### Overview

A simple alternative to https://github.com/andrewjfreyer/monitor which I failed to setup on my old MacMini 2010. This application performs a ping of given ips and reports node online/offline status via mqtt. Used in home automation scenarious when its required to distingush user presence/absence at home.

### Mqtt Api

- To receive statuses - subscribe to `device-pinger/<ip>/status` or wildcard `device-pinger/`, payload would be `{"online":true}` or `{"online":false}`
- Add new IP to monitor - publish to `device-pinger/<ip>/add` with any payload
- Delete IP from monitoring - publish to `device-pinger/<ip>/del` with any payload

### Configuration

Configuration is set via .env files. In development file .env is always loaded. In production there several options: use `make docker-run` to load default .env file, use `CONF=.env.sample make docker-run` to run with selected file, load envs from common docker-compose config from https://github.com/fedulovivan/mhz19-next/blob/master/.env.sample

### Production

- build image with `make docker-build`
- run image with default config `make docker-run`
- run image with any selected config `CONF=.env.sample make docker-run`

### Development

- build local binary `make`
- run `make run`

### Pitfalls

Application is terminated if no target IP is set or all were deleted, to be fixed.


### Screenshots

Console
![console.png](assets/02-console.png)
MQTT Explorer
![mqtt-explorer.png](assets/01-mqtt-explorer.png) 
Image size
![image-size.png](assets/03-image-size.png)
