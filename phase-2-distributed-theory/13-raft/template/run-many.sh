#!/usr/bin/env bash
#
# run-many.sh — run `go test -run <pattern>` many times in a row (or in
# parallel) to catch intermittent failures, the kind that distributed
# consensus code produces far more often than it should. Save the full
# output of every failing run to a directory, and print a pass/fail summary
# at the end.
#
# Usage:
#   run-many.sh -r PATTERN -n COUNT [-p PARALLEL] [-d LOGDIR] [-- go test args...]
#
# Options:
#   -r PATTERN   Regex passed to `go test -run` (required, e.g. "2A").
#   -n COUNT     Number of times to run the test (required, positive integer).
#   -p PARALLEL  How many runs to execute concurrently (default: 1, sequential).
#   -d LOGDIR    Directory to save failing runs' logs into (default: ./run-many-logs).
#   -h           Show this help and exit.
#
# Anything after a literal "--" is passed through to `go test` as additional
# arguments (for example: -race, -v, -count=1).
#
# Example:
#   ./run-many.sh -r 2A -n 50 -p 4 -d ./logs-2a -- -race

set -euo pipefail

usage() {
	sed -n '2,23p' "$0" | sed 's/^# \{0,1\}//'
}

pattern=""
count=""
parallel=1
logdir="./run-many-logs"

while getopts ":r:n:p:d:h" opt; do
	case "$opt" in
	r) pattern="$OPTARG" ;;
	n) count="$OPTARG" ;;
	p) parallel="$OPTARG" ;;
	d) logdir="$OPTARG" ;;
	h)
		usage
		exit 0
		;;
	\?)
		echo "error: unknown option -$OPTARG" >&2
		usage >&2
		exit 2
		;;
	:)
		echo "error: option -$OPTARG requires an argument" >&2
		usage >&2
		exit 2
		;;
	esac
done
shift $((OPTIND - 1))

# Everything left on the command line (after a "--" or otherwise) is passed
# through to `go test`.
extra_args=("$@")

if [ -z "$pattern" ]; then
	echo "error: -r PATTERN is required" >&2
	usage >&2
	exit 2
fi
if [ -z "$count" ]; then
	echo "error: -n COUNT is required" >&2
	usage >&2
	exit 2
fi
if ! [[ "$count" =~ ^[0-9]+$ ]] || [ "$count" -lt 1 ]; then
	echo "error: -n COUNT must be a positive integer, got '$count'" >&2
	exit 2
fi
if ! [[ "$parallel" =~ ^[0-9]+$ ]] || [ "$parallel" -lt 1 ]; then
	echo "error: -p PARALLEL must be a positive integer, got '$parallel'" >&2
	exit 2
fi

mkdir -p "$logdir"

pass_count=0
fail_count=0
running=0
declare -a pids=()
declare -a pid_runs=()

wait_for_one() {
	# Wait for exactly one background job to finish, record its result, and
	# remove it from the tracking arrays.
	local wait_pid
	wait_pid="${pids[0]}"
	local run_id="${pid_runs[0]}"
	pids=("${pids[@]:1}")
	pid_runs=("${pid_runs[@]:1}")

	local status=0
	wait "$wait_pid" || status=$?

	if [ "$status" -eq 0 ]; then
		pass_count=$((pass_count + 1))
		rm -f "$logdir/run-$run_id.log"
	else
		fail_count=$((fail_count + 1))
		echo "run $run_id: FAILED (exit $status), log saved to $logdir/run-$run_id.log"
	fi
	running=$((running - 1))
}

echo "running: go test -run '$pattern' ${extra_args[*]:-} (x$count, parallel=$parallel)"
echo "failing logs will be saved under: $logdir"

for ((i = 1; i <= count; i++)); do
	log_file="$logdir/run-$i.log"
	(go test -run "$pattern" "${extra_args[@]}" >"$log_file" 2>&1) &
	pids+=("$!")
	pid_runs+=("$i")
	running=$((running + 1))

	if [ "$running" -ge "$parallel" ]; then
		wait_for_one
	fi
done

while [ "$running" -gt 0 ]; do
	wait_for_one
done

echo ""
echo "summary: $pass_count passed, $fail_count failed (of $count total)"

if [ "$fail_count" -gt 0 ]; then
	exit 1
fi
