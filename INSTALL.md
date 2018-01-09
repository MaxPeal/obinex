# Obtaining and building obinex

Check out the obinex sources from Git into `$GOPATH/src`, along with all
dependencies:
```
go get gitlab.cs.fau.de/i4/obinex
go get golang.org/x/net/websocket golang.org/x/sys/unix
```
Now build the obinex binaries and put them into `$GOPATH/bin`:
```
go install gitlab.cs.fau.de/i4/obinex/...
```

# Installing obinex

Use `scripts/update.sh`.
