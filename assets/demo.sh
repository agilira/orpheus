#!/usr/bin/env bash
# demo.sh: the session recorded in assets/demo.gif
#
# Regenerate (asciinema 2.x and agg):
#   asciinema rec --cols 104 --rows 25 -c assets/demo.sh demo.cast
#   agg --theme monokai --font-size 16 --idle-time-limit 3 demo.cast assets/demo.gif
#
# 104x25 renders close to 16:9.

set -e
repo="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

(cd "$repo/examples/gitlike" && go build -o "$work/gitlike" .)
cd "$work"
export PATH="$work:$PATH" HOME="$work" # gitlike keeps its state in ~/.gitlike-config.json

say() { printf '\033[38;5;244m# %s\033[0m\n' "$1"; sleep 0.8; }

run() {
	printf '\033[1;32m$\033[0m '
	for ((i = 0; i < ${#1}; i++)); do
		printf '%s' "${1:i:1}"
		sleep 0.035
	done
	sleep 0.4
	echo
	eval "$1" || true
	echo
	sleep "${2:-1.8}"
}

clear
say "orpheus: a git-like CLI, commands and subcommands generated from the definitions"
run "gitlike --help" 3.5

clear
say "nested subcommands, short and long flags in any position"
run "gitlike remote add -f upstream https://github.com/agilira/orpheus.git"
run "gitlike remote list -v"
run "gitlike config set --global user.name Antonio" 1.2
run "gitlike status -sb" 2.5

clear
say "errors name the problem and carry the hints the handler attached"
run "gitlike stauts" 1.5
run "gitlike remote add origin" 2
run "gitlike remote add origin https://github.com/agilira/orpheus.git" 3
