#!/usr/bin/env bash
# Sends requests to the example server (go run ./examples/httpserver) and prints what came back.
# Usage: ./examples/httpserver/demo.sh [base-url] [timeline-csv]
set -euo pipefail
base=${1:-http://127.0.0.1:8080}
csv=${2:-}

start=$(date +%s.%N)
elapsed() { awk -v s="$start" -v n="$(date +%s.%N)" 'BEGIN { printf "%5.1fs", n - s }'; }

login() {
  local out code body
  out=$(curl -s -w '\n%{http_code}' -X POST "$base/login")
  code=${out##*$'\n'}
  body=${out%$'\n'*}
  body=${body%$'\n'}
  printf '%s  #%d  %s  %s\n' "$(elapsed)" "$1" "$code" "$body"
}

echo "== POST /login from one IP: a burst of 3, then one more every 2s"
for i in 1 2 3 4 5; do
  login "$i"
done
echo "   ... waiting 2.1s for one token to refill"
sleep 2.1
for i in 6 7; do
  login "$i"
done

sleep 3 # let the global bucket fill up again
echo
echo "== GET /hello every 100ms for 5s: global limit, a burst of 5, then one every 500ms"
start=$(date +%s.%N)
[ -n "$csv" ] && echo "ms,status" > "$csv"
line=""
from=0
for i in $(seq 0 49); do
  code=$(curl -s -o /dev/null -w '%{http_code}' "$base/hello")
  now=$(date +%s.%N)
  ms=$(awk -v s="$start" -v n="$now" 'BEGIN { printf "%d", (n - s) * 1000 }')
  [ -n "$csv" ] && echo "$ms,$code" >> "$csv"
  line+="$code "
  if (( (i + 1) % 10 == 0 )); then
    ok=$(grep -o 200 <<<"$line" | wc -l)
    printf '  %4.1fs-%4.1fs  %s (%2d of 10 allowed)\n' "$(awk -v m="$from" 'BEGIN { print m / 1000 }')" \
      "$(awk -v m="$ms" 'BEGIN { print m / 1000 }')" "$line" "$ok"
    line=""
    from=$ms
  fi
  sleep 0.1
done
