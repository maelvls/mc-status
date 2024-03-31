# Minecrat Status Server

The `mc-status` CLI lets you know if your minecraft server is up. It also lets
you create a server that shows that information.

## Deploy

```bash
KO_DOCKER_REPO=ghcr.io/maelvls/mc-status KO_DEFAULTBASEIMAGE=alpine \
  ko build . --bare --tarball /tmp/out.tar --push=false
ssh remote /usr/local/bin/docker load </tmp/out.tar
ssh remote sh -lc bin/deploy-mc-status
```

with `bin/deploy-mc-status`:

```bash
docker container inspect mc-status >/dev/null 2>/dev/null && docker rm -f mc-status || true
docker run -d --restart=always --name mc-status -p 8081:8081 \
  serve -mc-host=foo
```
