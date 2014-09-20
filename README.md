# gosub

`$GOPATH` submodule automation.

## usage

Synchronize modules in `$GOPATH` `.`, registering configuration in `.gitmodules`.

```sh
$ gosub -m .gitmodules -p . [packages...]
```

Or:

```sh
$ gosub [packages...]
```
