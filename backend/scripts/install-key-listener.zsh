#!/bin/zsh

function self-insert() {
    if [[ -z "$KEY_LISTENER_OUTPUT" ]]; then
        zle .self-insert
        return 0
    fi
    pwd=$(pwd)

    fieldsep="\0\e"
    endsep="\0\a\e"

    if [[ -n "$KEY_LISTENER_DEBUG" ]]; then
        fieldsep=" "
        endsep=" \n"
    fi

    zle .self-insert
    $ZDOTDIR/zkeylis -url "$KEY_LISTENER_OUTPUT" -pos "$(get_pos)" -dir "$pwd" -buffer "$BUFFER" -lbuffer "$LBUFFER" -rbuffer "$RBUFFER"

    # zle .self-insert
}

zle -N self-insert

function get_pos(){
  echo -ne "\033[6n" > /dev/tty
  read -t 1 -s -d 'R' pos < /dev/tty
  pos="${pos##*\[}"
  echo "$pos"
  # pos="${pos%;*}"
  # echo "$pos"
}
