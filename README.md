# SpoutMC

```
     =()=        
 ,/'\_||_           _____                   __  __  _________    
 ( (___  `.        / ___/____  ____  __  __/ /_/  |/  / ____/    
 `\./  `=='        \__ \/ __ \/ __ \/ / / / __/ /|_/ / /     
        |||       ___/ / /_/ / /_/ / /_/ / /_/ /  / / /___   
        |||      /____/ .___/\____/\__,_/\__/_/  /_/\____/   
        |||          /_/                            0.0.1       
```

Docker based Minecraft Server Networks

### Check
https://pkg.go.dev/github.com/docker/docker/client#pkg-overview  
https://docker-minecraft-server.readthedocs.io/en/latest/  
https://www.jsonschemavalidator.net/  
https://github.com/docker/cli/blob/master/cli/command/container/attach.go  
https://ourcodeworld.com/articles/read/1293/how-to-retrieve-the-skin-of-a-minecraft-user-from-mojang-using-python-3  
https://github.com/lus/mojango  
https://papermc.io/software/velocity  
https://pkg.go.dev/github.com/Tnze/go-mc  
https://hackernoon.com/the-essential-guide-to-load-balancing-minecraft-servers-with-kong-gateway-mr3i37aw  
https://github.com/WorldQL/worldql_server/tree/trunk  
https://github.com/WiIIiam278/HuskSync/tree/master  
https://github.com/Quutio/Bouncer  
https://github.com/Pumpkin-MC   
https://docs.docker.com/reference/api/engine/version/v1.50/


### Container Ideas
Connect to a Container in Docker without exposed ports
```
docker run --rm -p 3306:1234 --network spoutnetwork verb/socat TCP-LISTEN:1234,fork TCP-CONNECT:172.18.0.2:3306
```
Check this one out:  
https://stackoverflow.com/questions/19897743/exposing-a-port-on-a-live-docker-container/42071577
