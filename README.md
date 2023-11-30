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

https://docs.docker.com/engine/api/sdk/examples/#run-a-container-in-the-background  
https://pkg.go.dev/github.com/docker/docker/client#pkg-overview  
https://docker-minecraft-server.readthedocs.io/en/latest/  
https://towardsdatascience.com/use-environment-variable-in-your-next-golang-project-39e17c3aaa66  
https://betterstack.com/community/guides/logging/go/zap/  
https://www.jsonschemavalidator.net/  
https://github.com/docker/cli/blob/master/cli/command/container/attach.go

### Quickstart (Linux/MacOS/Windows)

Please find your compiled binaries in the repository's [Releases](https://github.com/0x49b/spout/releases)
section
Once, downloaded you may run the binaries & open `http://localhost:8081` to check the web UI

### Docker quickstart

```sh
mkdir data
docker run -p 8081:8081 -v ./data:/root/.config/spout -d pacholoamit/go-task-mgr:latest
```

### Docker-compose quickstart

```sh
mkdir spout
cd spout
curl https://raw.githubusercontent.com/pacholoamit/GO-TASK-MGR/master/docker-compose.yml >> docker-compose.yml
docker-compose up -d
```

Optionally, if you'd want to create your own `docker-compose.yml` file...

```yaml
# docker-compose.yml
---
version: "3.0"

services:
  task-mgr:
    container_name: spout
    image: pacholoamit/go-task-mgr:latest
    environment:
      PORT: 8081 # Optional: Default is 8081, sets the server port of the container.
    volumes:
      - ./data:/root/.config/spout
    ports:
      - 8081:8081
```

Then run

```bash
docker-compose up -d
```

