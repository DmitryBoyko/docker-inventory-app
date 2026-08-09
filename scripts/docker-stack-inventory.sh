#!/usr/bin/env bash
# Инвентаризация Docker-контейнеров по compose-стекам (аналог docker-stack-inventory.ps1).
# Вызовы: ps --size, stats, inspect, system df -v. Без N+1.
# Зависимости: bash 4+, docker, jq, column.
set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
  echo "docker не найден" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JQ_BIN=""
if command -v jq >/dev/null 2>&1; then
  JQ_BIN="$(command -v jq)"
elif [[ -x "$SCRIPT_DIR/jq" ]]; then
  JQ_BIN="$SCRIPT_DIR/jq"
elif [[ -f "$SCRIPT_DIR/jq.exe" ]]; then
  JQ_BIN="$SCRIPT_DIR/jq.exe"
fi
if [[ -z "$JQ_BIN" ]]; then
  echo "нужен jq (положите jq.exe рядом со скриптом или: apt/dnf install jq)" >&2
  exit 1
fi
jq() { "$JQ_BIN" "$@"; }

SPIN_PID=""
spin_start() {
  local msg="$1"
  (
    local frames='|/-\' i=0
    while true; do
      printf '\r%s %s   ' "$msg" "${frames:i++%${#frames}:1}" >&2
      sleep 0.12
    done
  ) &
  SPIN_PID=$!
}
spin_update() {
  local msg="$1"
  if [[ -n "${SPIN_PID}" ]]; then
    kill "$SPIN_PID" 2>/dev/null || true
    SPIN_PID=""
  fi
  printf '\r\033[K' >&2
  spin_start "$msg"
}
spin_stop() {
  if [[ -n "${SPIN_PID}" ]]; then
    kill "$SPIN_PID" 2>/dev/null || true
    SPIN_PID=""
  fi
  printf '\r\033[K' >&2
}

size_to_bytes() {
  local t="${1:-}"
  t="${t// /}"
  [[ -z "$t" ]] && { echo 0; return; }
  local re='^([0-9]+([.][0-9]+)?)([kKmMgGtT]i?[bB]|[bB])$'
  if [[ "$t" =~ $re ]]; then
    local v="${BASH_REMATCH[1]}"
    local u
    u="$(printf '%s' "${BASH_REMATCH[3]}" | tr '[:upper:]' '[:lower:]')"
    # bash arithmetic is integer — scale via awk only once for float*factor is still ok,
    # but keep one awk call; for speed use integer approx when no dot
    case "$u" in
      b)   printf '%d\n' "${v%.*}" ;;
      kb)  awk -v v="$v" 'BEGIN{printf "%d", v*1000}' ;;
      mb)  awk -v v="$v" 'BEGIN{printf "%d", v*1000*1000}' ;;
      gb)  awk -v v="$v" 'BEGIN{printf "%d", v*1000*1000*1000}' ;;
      tb)  awk -v v="$v" 'BEGIN{printf "%d", v*1000*1000*1000*1000}' ;;
      kib) awk -v v="$v" 'BEGIN{printf "%d", v*1024}' ;;
      mib) awk -v v="$v" 'BEGIN{printf "%d", v*1024*1024}' ;;
      gib) awk -v v="$v" 'BEGIN{printf "%d", v*1024*1024*1024}' ;;
      tib) awk -v v="$v" 'BEGIN{printf "%d", v*1024*1024*1024*1024}' ;;
      *)   printf '%d\n' "${v%.*}" ;;
    esac
  else
    echo 0
  fi
}

format_bytes() {
  local b="${1:-0}"
  awk -v b="$b" 'BEGIN {
    if (b+0 >= 1073741824) printf "%.2f GB", b/1073741824
    else if (b+0 >= 1048576) printf "%.2f MB", b/1048576
    else if (b+0 >= 1024) printf "%.2f KB", b/1024
    else printf "%d B", b+0
  }'
}

format_vol_name() {
  local n="$1"
  local re='^[a-f0-9]{64}$'
  if [[ "$n" =~ $re ]]; then
    echo "anon:${n:0:12}..."
  else
    echo "$n"
  fi
}

short_image() {
  local img="${1:-}"
  [[ -z "$img" || "$img" == "null" ]] && { echo "-"; return; }
  local re='^sha256:([a-f0-9]{12})'
  if [[ "$img" =~ $re ]]; then
    echo "sha256:${BASH_REMATCH[1]}..."
  else
    echo "$img"
  fi
}

# prints: EXTERNAL<TAB>INTERNAL
split_ports() {
  local ports="${1:-}"
  if [[ -z "$ports" || "$ports" == "null" ]]; then
    printf '%s\t%s\n' '-' '-'
    return
  fi

  declare -A ext_ips=()
  declare -a ext_keys=()
  declare -a int_ports=()
  declare -A int_seen=()

  local p left right hostIp hostPort dest key ip allIfaces onlyLocal extra piece external internal x
  local rest="$ports"
  while [[ -n "$rest" ]]; do
    if [[ "$rest" == *,* ]]; then
      p="${rest%%,*}"
      rest="${rest#*,}"
    else
      p="$rest"
      rest=""
    fi
    # trim spaces
    p="${p#"${p%%[![:space:]]*}"}"
    p="${p%"${p##*[![:space:]]}"}"
    [[ -z "$p" ]] && continue

    if [[ "$p" == *"->"* ]]; then
      left="${p%%->*}"
      dest="${p#*->}"
      # работает и для 0.0.0.0:15173, и для [::]:15173
      if [[ "$left" == *:* ]]; then
        hostIp="${left%:*}"
        hostPort="${left##*:}"
      else
        hostIp=""
        hostPort=""
      fi
      if [[ -n "$hostPort" && -n "$dest" ]]; then
        key="${hostPort}->${dest}"
        if [[ -z "${ext_ips[$key]+x}" ]]; then
          ext_keys+=("$key")
          ext_ips[$key]="$hostIp"
        else
          case ";${ext_ips[$key]};" in
            *";$hostIp;"*) ;;
            *) ext_ips[$key]="${ext_ips[$key]};$hostIp" ;;
          esac
        fi
      else
        key="$p"
        if [[ -z "${ext_ips[$key]+x}" ]]; then
          ext_keys+=("$key")
          ext_ips[$key]=""
        fi
      fi
    else
      if [[ -z "${int_seen[$p]+x}" ]]; then
        int_seen[$p]=1
        int_ports+=("$p")
      fi
    fi
  done

  external=""
  for key in "${ext_keys[@]}"; do
    local ips="${ext_ips[$key]}"
    piece=""
    if [[ -z "$ips" ]]; then
      piece="$key"
    else
      allIfaces=0
      onlyLocal=1
      extra=0
      local OLDIFS="$IFS"
      IFS=';'
      for ip in $ips; do
        case "$ip" in
          0.0.0.0|'*'|'[::]') allIfaces=1 ;;
        esac
        case "$ip" in
          127.0.0.1|'[::1]') ;;
          *) onlyLocal=0 ;;
        esac
        case "$ip" in
          0.0.0.0|'*'|'[::]'|127.0.0.1|'[::1]') ;;
          *) extra=1 ;;
        esac
      done
      if ((allIfaces)) && ((extra==0)) && ((onlyLocal==0)); then
        piece="*:${key} [наружу]"
      elif ((onlyLocal)); then
        piece="127.0.0.1:${key} [localhost]"
      else
        piece=""
        for ip in $ips; do
          [[ -n "$piece" ]] && piece+='; '
          piece+="${ip}:${key}"
        done
      fi
      IFS="$OLDIFS"
    fi
    [[ -n "$external" ]] && external+=' | '
    external+="$piece"
  done

  internal=""
  for x in "${int_ports[@]+"${int_ports[@]}"}"; do
    [[ -n "$internal" ]] && internal+='; '
    internal+="$x"
  done

  [[ -z "$external" ]] && external='-'
  [[ -z "$internal" ]] && internal='-'
  printf '%s\t%s\n' "$external" "$internal"
}

YELLOW=$'\033[33m'
CYAN=$'\033[36m'
MAGENTA=$'\033[35m'
GREEN=$'\033[32m'
GRAY=$'\033[90m'
DARKCYAN=$'\033[36m'
RESET=$'\033[0m'

TMPDIR_RUN="$(mktemp -d)"
cleanup() {
  spin_stop
  rm -rf "$TMPDIR_RUN"
}
trap cleanup EXIT

PS_JSON="$TMPDIR_RUN/ps.jsonl"
STATS_JSON="$TMPDIR_RUN/stats.jsonl"
INSPECT_JSON="$TMPDIR_RUN/inspect.json"
VOL_MAP="$TMPDIR_RUN/vol.map"
ROWS="$TMPDIR_RUN/rows.tsv"

spin_start "Сбор: контейнеры..."
docker ps -a --size --format '{{json .}}' >"$PS_JSON"

spin_update "Сбор: stats..."
docker stats --no-stream --format '{{json .}}' >"$STATS_JSON" 2>/dev/null || true

spin_update "Сбор: inspect..."
mapfile -t IDS < <(docker ps -aq)
if ((${#IDS[@]} > 0)); then
  docker inspect "${IDS[@]}" >"$INSPECT_JSON"
else
  echo '[]' >"$INSPECT_JSON"
fi

spin_update "Сбор: размеры томов..."
: >"$VOL_MAP"
in_vol=0
while IFS= read -r line || [[ -n "$line" ]]; do
  re_volhdr='VOLUME[[:space:]]+NAME[[:space:]]+LINKS[[:space:]]+SIZE'
  if [[ "$line" =~ $re_volhdr ]]; then
    in_vol=1
    continue
  fi
  if ((in_vol)); then
    [[ -z "${line// /}" ]] && break
    re_bcache='Build[[:space:]]+cache'
    [[ "$line" =~ $re_bcache ]] && break
    # NAME LINKS SIZE
    read -r vname vlinks vsize _ <<<"$line" || true
    if [[ -n "${vname:-}" && -n "${vlinks:-}" && -n "${vsize:-}" && "$vlinks" != *[!0-9]* ]]; then
      printf '%s\t%s\t%s\t%s\n' "$vname" "$vlinks" "$vsize" "$(size_to_bytes "$vsize")" >>"$VOL_MAP"
    fi
  fi
done < <(docker system df -v 2>/dev/null || true)

spin_update "Обработка..."

STATS_BY_ID="$TMPDIR_RUN/stats_by_id.json"
if [[ -s "$STATS_JSON" ]]; then
  jq -s 'map(select(.ID != null) | {key: .ID, value: .}) | from_entries' "$STATS_JSON" >"$STATS_BY_ID"
else
  echo '{}' >"$STATS_BY_ID"
fi

INSPECT_BY_ID="$TMPDIR_RUN/inspect_by_id.json"
jq '
  map({
    key: (.Id[0:12]),
    value: {
      volumes: [.Mounts[]? | select(.Type=="volume" and (.Name|type=="string")) | .Name],
      networks: (.NetworkSettings.Networks // {} | keys),
      ips: [
        .NetworkSettings.Networks // {} | to_entries[]
        | select((.value.IPAddress // "") != "")
        | "\(.value.IPAddress) (\(.key))"
      ],
      restarts: (.RestartCount // 0),
      health: (if .State.Health then .State.Health.Status else "-" end)
    }
  }) | from_entries
' "$INSPECT_JSON" >"$INSPECT_BY_ID"

# Один проход jq -> записи с разделителем US (\x1f), без jq в цикле
# Поля: stack container service image ports status state running_for size mem cpu net_io block_io volnames networks ips restarts health
FS=$'\x1f'
jq -r --slurpfile stats "$STATS_BY_ID" --slurpfile insp "$INSPECT_BY_ID" '
  . as $c
  | ($stats[0][$c.ID] // null) as $st
  | ($insp[0][$c.ID] // {
      volumes: [], networks: [], ips: [], restarts: 0, health: "-"
    }) as $i
  | ($c.Labels // "") as $labels
  | [
      (if ($labels | test("com\\.docker\\.compose\\.project="))
       then ($labels | capture("com\\.docker\\.compose\\.project=(?<p>[^,]+)") | .p)
       else "standalone" end),
      ($c.Names // "-"),
      (if ($labels | test("com\\.docker\\.compose\\.service="))
       then ($labels | capture("com\\.docker\\.compose\\.service=(?<s>[^,]+)") | .s)
       else "-" end),
      ($c.Image // "-"),
      (($c.Ports // "") | gsub("\r"; "")),
      (($c.Status // "") | gsub("\r"; "")),
      (($c.State // "-") | gsub("\r"; "")),
      (($c.RunningFor // "-") | gsub("\r"; "")),
      (($c.Size // "0B") | gsub("\r"; "")),
      (if $st then (($st.MemUsage | split(" / ")[0]) | gsub("\r"; "")) else "Stopped" end),
      (if $st then (($st.CPUPerc // "-") | gsub("\r"; "")) else "-" end),
      (if $st then (($st.NetIO // "-") | gsub("\r"; "")) else "-" end),
      (if $st then (($st.BlockIO // "-") | gsub("\r"; "")) else "-" end),
      (($i.volumes // []) | join("|")),
      (if (($i.networks // [])|length) == 0 then "-" else ($i.networks | join(", ")) end),
      (if (($i.ips // [])|length) == 0 then "-" else ($i.ips | join("; ")) end),
      ($i.restarts // 0 | tostring),
      (($i.health // "-") | gsub("\r"; ""))
    ]
  | map(tostring | gsub("\u001f"; " "))
  | join("\u001f")
' "$PS_JSON" >"$TMPDIR_RUN/raw.usv"

strip_cr() { printf '%s' "${1//$'\r'/}"; }

: >"$ROWS"
while IFS=$'\x1f' read -r stack container service image ports_raw status state running_for size_raw mem cpu net_io block_io vol_names nets ips restarts health || [[ -n "${stack:-}" ]]; do
  [[ -z "${stack:-}" ]] && continue
  stack="$(strip_cr "$stack")"
  container="$(strip_cr "$container")"
  service="$(strip_cr "$service")"
  image="$(short_image "$(strip_cr "$image")")"
  ports_raw="$(strip_cr "$ports_raw")"
  status="$(strip_cr "$status")"
  state="$(strip_cr "$state")"
  running_for="$(strip_cr "$running_for")"
  size_raw="$(strip_cr "$size_raw")"
  mem="$(strip_cr "$mem")"
  cpu="$(strip_cr "$cpu")"
  net_io="$(strip_cr "$net_io")"
  block_io="$(strip_cr "$block_io")"
  vol_names="$(strip_cr "$vol_names")"
  nets="$(strip_cr "$nets")"
  ips="$(strip_cr "$ips")"
  restarts="$(strip_cr "$restarts")"
  health="$(strip_cr "$health")"

  disk="${size_raw%% (virtual*}"
  disk="${disk%"${disk##*[![:space:]]}"}"
  disk_bytes="$(size_to_bytes "$disk")"

  mem_bytes=0
  if [[ "$mem" != "Stopped" ]]; then
    mem_bytes="$(size_to_bytes "$mem")"
  fi

  cpu_val="${cpu%%%}"
  cpu_val="${cpu_val%\%}"
  re_num='^[0-9]+([.][0-9]+)?$'
  [[ "$cpu_val" =~ $re_num ]] || cpu_val=0

  if [[ "$health" == "-" || "$health" == "null" || -z "$health" ]]; then
    case "$status" in
      *'(healthy)'*) health='healthy' ;;
      *'(unhealthy)'*) health='unhealthy' ;;
      *'(health: starting)'*|*'(starting)'*) health='starting' ;;
      *) health='-' ;;
    esac
  fi

  uptime="$running_for"
  if [[ "$state" == "exited" || "$state" == "created" ]]; then
    uptime="$status"
  fi

  ports_line="$(split_ports "$ports_raw")"
  external="$(strip_cr "${ports_line%%$'\t'*}")"
  internal="$(strip_cr "${ports_line#*$'\t'}")"

  vol_display="-"
  if [[ -n "$vol_names" ]]; then
    vol_display=""
    IFS='|' read -r -a _vols <<<"$vol_names"
    for v in "${_vols[@]}"; do
      [[ -z "$v" ]] && continue
      [[ -n "$vol_display" ]] && vol_display+=", "
      vol_display+="$(format_vol_name "$v")"
    done
    [[ -z "$vol_display" ]] && vol_display="-"
  fi

  # ROWS: 1stack 2container 3service 4image 5ext 6int 7ip 8nets 9voldisp 10volnames
  # 11cpu 12cpuval 13mem 14membytes 15netio 16blockio 17disk 18diskbytes 19health 20restarts 21uptime 22state
  printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
    "$stack" "$container" "$service" "$image" "$external" "$internal" "$ips" "$nets" \
    "$vol_display" "$vol_names" "$cpu" "$cpu_val" "$mem" "$mem_bytes" \
    "$net_io" "$block_io" "$disk" "$disk_bytes" "$health" "$restarts" "$uptime" "$state" \
    >>"$ROWS"
done <"$TMPDIR_RUN/raw.usv"

spin_stop

if [[ ! -s "$ROWS" ]]; then
  echo "Контейнеров нет."
  exit 0
fi

# убрать CR из готовой таблицы (на всякий случай)
sed -i 's/\r$//' "$ROWS" 2>/dev/null || sed -i '' 's/\r$//' "$ROWS" 2>/dev/null || true

mapfile -t STACKS < <(cut -f1 "$ROWS" | sort -u)

grand_disk=0
grand_mem=0
grand_cpu="0"
declare -A SEEN_VOL=()
all_vol_bytes=0
all_count=0

vol_lookup() {
  local name="$1"
  local line
  line="$(awk -F'\t' -v n="$name" '$1==n { print $2 "\t" $3 "\t" $4; exit }' "$VOL_MAP")"
  if [[ -z "$line" ]]; then
    printf '%s\t%s\t%s\n' '?' '?' '0'
  else
    printf '%s\n' "$line"
  fi
}

for stack in "${STACKS[@]}"; do
  echo
  printf '%s=== STACK: %s ===%s\n' "$YELLOW" "$stack" "$RESET"

  {
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
      Container Service Image Health State Restarts Uptime CPU Memory Disk
    awk -F'\t' -v s="$stack" '$1==s {
      print $14 "\t" $2 "\t" $3 "\t" $4 "\t" $19 "\t" $22 "\t" $20 "\t" $21 "\t" $11 "\t" $13 "\t" $17
    }' "$ROWS" | sort -t$'\t' -k1,1nr | cut -f2-
  } | column -t -s $'\t' || true
  echo

  {
    printf '%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
      Container External Internal IP Networks Volumes NetIO BlockIO
    awk -F'\t' -v s="$stack" '$1==s {
      print $2 "\t" $5 "\t" $6 "\t" $7 "\t" $8 "\t" $9 "\t" $15 "\t" $16
    }' "$ROWS" | sort -t$'\t' -k1,1
  } | column -t -s $'\t' || true
  echo

  cnt=0; with_vols=0; unhealthy=0; restarting=0; total_disk=0; total_mem=0; total_cpu=0
  read -r cnt with_vols unhealthy restarting total_disk total_mem total_cpu < <(
    awk -F'\t' -v s="$stack" '
      $1==s {
        c++
        if ($9 != "-") wv++
        if ($19 ~ /unhealthy/) uh++
        if (($20+0) > 0) rs++
        td += $18+0
        if ($22 == "running") { tm += $14+0; tc += $12+0 }
      }
      END { printf "%d %d %d %d %d %d %.4f", c+0, wv+0, uh+0, rs+0, td+0, tm+0, tc+0 }
    ' "$ROWS"
  ) || true
  cnt=${cnt:-0}
  with_vols=${with_vols:-0}
  unhealthy=${unhealthy:-0}
  restarting=${restarting:-0}
  total_disk=${total_disk:-0}
  total_mem=${total_mem:-0}
  total_cpu=${total_cpu:-0}

  printf '%sContainers: %s  |  with volumes: %s  |  unhealthy: %s  |  restarts>0: %s%s\n' \
    "$GRAY" "$cnt" "$with_vols" "$unhealthy" "$restarting" "$RESET"
  printf '%sStack RAM (running): %s  |  Stack CPU: %s%%%s\n' \
    "$CYAN" "$(format_bytes "$total_mem")" "$total_cpu" "$RESET"
  printf '%sStack Disk writable: %s%s\n' \
    "$CYAN" "$(format_bytes "$total_disk")" "$RESET"

  top_text="$(
    awk -F'\t' -v s="$stack" '$1==s && ($14+0)>0 { print $14 "\t" $2 "\t" $13 }' "$ROWS" \
      | sort -t$'\t' -k1,1nr \
      | head -n 3 \
      | awk -F'\t' '{ if (NR>1) printf "  |  "; printf "%s (%s)", $2, $3 }'
  )" || true
  if [[ -n "$top_text" ]]; then
    printf '%sTop RAM: %s%s\n' "$MAGENTA" "$top_text" "$RESET"
  fi

  mapfile -t stack_vols < <(
    awk -F'\t' -v s="$stack" '
      $1==s && $10 != "" {
        n = split($10, a, "|")
        for (i = 1; i <= n; i++) if (a[i] != "") print a[i]
      }
    ' "$ROWS" | sort -u
  ) || true

  if ((${#stack_vols[@]} > 0)); then
    printf '%sVolumes (stack):%s\n' "$DARKCYAN" "$RESET"
    stack_vol_bytes=0
    for vn in "${stack_vols[@]}"; do
      vn="$(strip_cr "$vn")"
      [[ -z "$vn" ]] && continue
      meta="$(vol_lookup "$vn")"
      links="$(printf '%s' "$meta" | cut -f1)"
      size="$(printf '%s' "$meta" | cut -f2)"
      bytes="$(printf '%s' "$meta" | cut -f3)"
      bytes=${bytes:-0}
      [[ "$bytes" != *[!0-9]* ]] || bytes=0
      stack_vol_bytes=$((stack_vol_bytes + bytes))
      printf '  - %s  |  %s  |  links=%s\n' "$(format_vol_name "$vn")" "$size" "$links"
      if [[ -z "${SEEN_VOL[$vn]+x}" ]]; then
        SEEN_VOL[$vn]=1
        all_vol_bytes=$((all_vol_bytes + bytes))
      fi
    done
    printf '%s  = Volume data total: %s%s\n' "$CYAN" "$(format_bytes "$stack_vol_bytes")" "$RESET"
  else
    printf '%sVolumes (stack): -%s\n' "$DARKCYAN" "$RESET"
  fi

  printf '%.0s-' {1..72}; echo

  grand_disk=$((grand_disk + total_disk))
  grand_mem=$((grand_mem + total_mem))
  grand_cpu="$(awk -v a="$grand_cpu" -v b="$total_cpu" 'BEGIN { printf "%.4f", a+b }')"
  all_count=$((all_count + cnt))
done

echo
printf '%sALL containers: %s%s\n' "$GREEN" "$all_count" "$RESET"
printf '%sALL RAM (running): %s  |  ALL CPU: %s%%%s\n' \
  "$GREEN" "$(format_bytes "$grand_mem")" "$grand_cpu" "$RESET"
printf '%sALL writable layers: %s%s\n' "$GREEN" "$(format_bytes "$grand_disk")" "$RESET"
printf '%sALL volume data (unique): %s%s\n' "$GREEN" "$(format_bytes "$all_vol_bytes")" "$RESET"
