# Whatsup-Poll
CLI Whatsapp Polls using go:
- [whatsmeow](https://github.com/tulir/whatsmeow), which now also has python bindings with [neonize](https://github.com/krypton-byte/neonize).
- [urfave/cli](https://github.com/urfave/cli) for the command line

# Usage
Can be easily scheduled using `crontab` because of its non-interactive operation
```
NAME:
   whatsup-poll - Run WhatsApp actions from your CLI. User JID has to end with '@s.whatsapp.net', Group ID with '@g.us'

USAGE:
   whatsup-poll [global options] [command [command options]]

COMMANDS:
   message    send a message using <JID> <MESSAGE>
   getgroups  print all available group info
   poll       send a poll using <JID> <HEADER> <OPTIONS> ; requires a group ID
   help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help
```
