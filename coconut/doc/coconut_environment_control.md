## coconut environment control

control the state machine of an environment

### Synopsis

The environment control command triggers an event in the state 
machine of an existing O² environment. The event, if valid, starts a transition. 
The reached state is returned.

An event name must be passed via the mandatory event flag.
Valid events:
  CONFIGURE            RESET                EXIT
  START_ACTIVITY       STOP_ACTIVITY

Not all events are available in all states.

```
coconut environment control [environment id] [flags]
```

### Options

```
  -e, --event string   environment state machine event to trigger
  -h, --help           help for control
```

### Options inherited from parent commands

```
      --config string            optional configuration file for coconut (default $HOME/.config/coconut/settings.yaml)
      --config_endpoint string   configuration endpoint used by AliECS core as PROTO://HOST:PORT (default "consul://127.0.0.1:8500")
      --endpoint string          AliECS core endpoint as HOST:PORT (default "127.0.0.1:47102")
  -v, --verbose                  show verbose output for debug purposes
```

### SEE ALSO

* [coconut environment](coconut_environment.md)	 - create, destroy and manage AliECS environments

###### Auto generated by spf13/cobra on 26-Aug-2019