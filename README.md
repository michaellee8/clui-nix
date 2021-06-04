# clui-nix

An experiement to provide a CLUI (Command Line User Interface) for a genercic 
Linux enviroment, such as shell and text editors.

## Motivation

Frequently I found myself trying to do some code editing inside a mobile device, 
something like a mobile phone or a tablet, so that I don't need to bring my 
laptop. However, it is not really convenient to do those on a mobile device, 
especially when you try to use neovim and sh from a remote ssh session, you 
are really lacking those completion functionalities you can get from a mobile 
keyboard (something like gboard), and trying to access completion functionalites 
provided by the cli programs itself is indeed hard. You will need to use arrow 
keys to navigate those completion options, or you will have to type them 
yourself. Both options are indeed hard to use in a touch-based device.

Then I came to a [blog](https://blog.replit.com/clui) from the Repl.it team 
describing an interesting UI pattern, which users type in text commands in a 
search box like what would you get in VSCode with Ctrl-Shift-P, and then the 
application has a completion engine which would suggest users with what they can 
type in, with a help message avilable for each commands. However their 
implementation is based on a tree of known syntax, which is specific to their 
application, and won't work with existing completion engines like `compsys` in 
zsh and YouCompleteMe in vim. 

That gap is what this experiment for, implmenting CLUI for existing 
terminal-based applications (e.g. zsh) by interfacing with existing 
completion engines (e.g. compsys) and then providing an interface for a 
terminal emulator (e.g. termux on android or xterm) to access these completion 
options, which they can type them in when the user select it, as well as, if 
possible, a help message for those completion options. Such a system will not 
only provide better UX for TUI applications on touch-based device, it will also 
allow new comers of Linux systems to learn and access the commands more easily.

## Proof of concept scope

- Support Linux only
- A Docker image which contains a Go program, which will in turn initiate a zsh 
  instance and capture its completion options, and then provide two binary 
  stream, the first one handles the raw terminal I/O and the second one provide 
  completion options. These streams are exposed through WebSocket.
- A Flutter library providing a POC implmenetation of the said CLUI, based on 
  xterm.dart, interfacing with the Golang container.
