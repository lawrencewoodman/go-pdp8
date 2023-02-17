# Fixtures

In order to test the emulator the fixtures/ directory needs a number of paper tape images.  Below we can see where to obtain them.

To download these images we can use cmd/downloadfixtures

To build the command from the root of the repo use:

```
$ go build ./cmd/downloadfixtures
```

We can then run it using the following to download the files needed to fixtures/ if we add the `-confirm` switch we are confirming that we are aware that the files may be copyrighted:

```
$ ./downloadfixtures -confirm fixtures
```

If you would rather download the files manually you can see them and their sources listed in cmd/downloadfixtures/main.go
