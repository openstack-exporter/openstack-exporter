#!/bin/bash -e

set_default() {
  variable="$(snapctl get $1)"
  snapctl set $1=${variable:-$2}
}

add_option() {
    key=$1
    value=$(snapctl get $key)
    if [ -n "$value" ]; then
        echo "--$key=$value"
    fi
}

add_argument() {
    key=$1
    value=$(snapctl get $key)
    if [ -n "$value" ]; then
        echo "$value"
    fi
}
