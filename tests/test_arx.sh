#!/usr/bin/env bash
# Minimal smoke test for arx. Source bin/arx, exercise the fixed paths.
# Run: bash tests/test_arx.sh   (from repo root)
set -uo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/.." && pwd)"
# shellcheck source=../bin/arx
source "$root/bin/arx"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
cd "$TMP"

pass=0; fail=0
ok()   { if [[ $1 -eq 0 ]]; then echo "  ok   $2"; pass=$((pass+1)); else echo "  FAIL $2 (rc=$1)"; fail=$((fail+1)); fi; }
bad()  { if [[ $1 -ne 0 ]]; then echo "  ok   $2"; pass=$((pass+1)); else echo "  FAIL $2 (expected nonzero)"; fail=$((fail+1)); fi; }

echo "arx version: $(arx --version 2>&1 | head -1)"

# T1 basic compress + extract roundtrip
mkdir -p data && echo hello > data/a.txt && echo log > data/b.log
arx -c tar.gz -n bk data/ >/dev/null 2>&1; ok $? "compress tar.gz"
[[ -f bk.tar.gz ]] || { echo "  FAIL bk.tar.gz missing"; fail=$((fail+1)); }
mkdir -p out && arx bk.tar.gz -t out >/dev/null 2>&1; ok $? "extract auto-detect"
[[ -f out/data/a.txt ]] || { echo "  FAIL extract content"; fail=$((fail+1)); }

# T2 jobs validation rejects garbage
arx -c tar.gz -j abc -n x data/ >/dev/null 2>&1; bad $? "jobs rejects non-int"
arx -c tar.gz -j 0 -n x data/ >/dev/null 2>&1; bad $? "jobs rejects zero"

# T3 secure password prompts require a TTY; automation must opt into argv exposure
printf 'secret\n' | arx -c zip -p -n sec data/ >/dev/null 2>&1; bad $? "password prompt rejects non-interactive input"
printf 'secret\n' | arx -c zip -p --allow-insecure-password -n sec data/ >/dev/null 2>&1; ok $? "explicit insecure password mode works"

# T4 convert tar.gz -> tar.zst roundtrip
arx convert bk.tar.gz to cv.tar.zst >/dev/null 2>&1; ok $? "convert tar.gz->tar.zst"
[[ -f cv.tar.zst ]] || { echo "  FAIL cv.tar.zst missing"; fail=$((fail+1)); }

# T5 pipefail guard: arx -c must fail (and not "Archive created") when output cannot be written
mkdir -p missing_input && echo data > missing_input/real.txt
arx -c tar.gz -n ro_test -t /root missing_input/ >/dev/null 2>&1
bad $? "pipefail: arx -c fails when output unwritable (no silent success)"
rm -rf missing_input

# T6 delete_after only on real success (use a good compress)
cp -r data data_del
arx -c tar.gz -n del -d data_del/ >/dev/null 2>&1
if [[ $? -eq 0 && ! -e data_del && -f del.tar.gz ]]; then echo "  ok   delete_after on success"; pass=$((pass+1)); else echo "  FAIL delete_after"; fail=$((fail+1)); fi

# T7 split-size validation and verify-before-split
arx -c tar.gz --split nonsense -n invalid_split data/ >/dev/null 2>&1; bad $? "split rejects invalid size"
arx -c tar.gz --verify --split 1K -n split_ok data/ >/dev/null 2>&1; ok $? "verify completes before split"
[[ -f split_ok.tar.gz.00 ]] || { echo "  FAIL split part missing"; fail=$((fail+1)); }

# T8 path traversal is rejected before extraction
python3 - <<'PYTEST'
import io
import tarfile

with tarfile.open("malicious.tar", "w") as archive:
    payload = b"escape"
    info = tarfile.TarInfo("../escape.txt")
    info.size = len(payload)
    archive.addfile(info, io.BytesIO(payload))
PYTEST
mkdir -p traversal_out
arx malicious.tar -t traversal_out >/dev/null 2>&1; bad $? "tar traversal is rejected"
[[ ! -e escape.txt ]] || { echo "  FAIL traversal escaped target"; fail=$((fail+1)); }

echo ""
echo "PASS=$pass FAIL=$fail"
[[ $fail -eq 0 ]]
