## Documentation

If you would like to see the MAINDEC and DEC documentation associated with the testing files you can download them with cmd/downloaddocs

To build the command from the root of the repo use:

```
$ go build ./cmd/downloaddocs
```

We can then run it using the following to download the files to 'docs/', if we add the `-confirm` switch we are confirming that we are aware that the files may be copyrighted:

```
$ ./downloaddocs -confirm docs/
```

If you would rather download the files manually you can see them and their sources listed in 'cmd/downloaddocs/main.go'
