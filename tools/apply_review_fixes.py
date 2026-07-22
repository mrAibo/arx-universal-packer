from pathlib import Path


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected one match, found {count}")
    return text.replace(old, new, 1)


path = Path("bin/arx")
text = path.read_text()

text = replace_once(
    text,
    '  dry_run=false no_confirm=false debug=false password="" allow_insecure_password=false verify_archive=false',
    '  dry_run=false no_confirm=false debug=false password="" password_requested=false allow_insecure_password=false verify_archive=false',
    "password defaults",
)

text = replace_once(
    text,
    '        -p|--password) log_warning "7z/zip only accept argv passwords"; read -rs -p "Password: " password; printf "\\n"; [[ -n "$password" ]] || { log_error "Password must not be empty"; return 1; }; shift ;;',
    '        -p|--password) password_requested=true; shift ;;',
    "password parsing",
)

jobs_validation = '''    if [[ -n "${jobs_set:-}" ]]; then
      [[ "$jobs" =~ ^[0-9]+$ && "$jobs" -ge 1 ]] || { log_error "Number of jobs must be a positive integer"; return 1; }
    fi
'''
extra_validation = '''    if [[ -n "${jobs_set:-}" ]]; then
      [[ "$jobs" =~ ^[0-9]+$ && "$jobs" -ge 1 ]] || { log_error "Number of jobs must be a positive integer"; return 1; }
    fi
    if [[ -n "${split_size:-}" ]]; then
      [[ "$split_size" =~ ^[1-9][0-9]*([KMGkmg])?$ ]] || { log_error "Invalid split size: $split_size (use bytes or a suffix such as 100M or 1G)"; return 1; }
      split_size="${split_size^^}"
      command -v split >/dev/null 2>&1 || { log_error "Missing dependency: split"; return 1; }
    fi
    if [[ "$password_requested" == "true" ]]; then
      if [[ "$mode" == "compress" && "$compress_format" != "zip" && "$compress_format" != "7z" ]]; then
        log_error "Password protection is supported only for zip and 7z"
        return 1
      fi
      if [[ "$mode" == "convert" || "$mode" == "list" ]]; then
        log_error "Password prompting is not supported for $mode mode"
        return 1
      fi
      if [[ "$allow_insecure_password" == "true" ]]; then
        log_warning "Password will be passed as a process argument and may be visible to other users"
        read -rs -p "Password: " password
        printf "\\n"
        [[ -n "$password" ]] || { log_error "Password must not be empty"; return 1; }
      elif [[ ! -t 0 || ! -t 1 ]]; then
        log_error "Secure password prompting requires an interactive terminal"
        printf "%s\\n" "Use --allow-insecure-password only for trusted automation." >&2
        return 1
      fi
    fi
'''
text = replace_once(text, jobs_validation, extra_validation, "argument validation")

text = replace_once(
    text,
    '        if [[ -n "$password" ]]; then zip_opts+=("-P" "$password"); fi',
    '''        if [[ "$password_requested" == "true" ]]; then
          if [[ "$allow_insecure_password" == "true" ]]; then zip_opts+=("-P" "$password"); else zip_opts+=("-e"); fi
        fi''',
    "zip creation password",
)
text = replace_once(
    text,
    '        if [[ -n "$password" ]]; then sz+=("-p${password}" "-mhe=on"); fi',
    '''        if [[ "$password_requested" == "true" ]]; then
          if [[ "$allow_insecure_password" == "true" ]]; then sz+=("-p${password}"); else sz+=("-p"); fi
          sz+=("-mhe=on")
        fi''',
    "7z creation password",
)

text = replace_once(
    text,
    '          if [[ -n "$password" ]]; then unzip "${unzip_opts[@]}" -P "$password" "$f" -d "$extract_target"; else unzip "${unzip_opts[@]}" "$f" -d "$extract_target"; fi',
    '''          if [[ "$password_requested" == "true" && "$allow_insecure_password" == "true" ]]; then
            unzip "${unzip_opts[@]}" -P "$password" "$f" -d "$extract_target"
          else
            unzip "${unzip_opts[@]}" "$f" -d "$extract_target"
          fi''',
    "zip extraction password",
)
text = replace_once(
    text,
    '          [[ -n "$password" ]] && sz+=("-p${password}")',
    '''          if [[ "$password_requested" == "true" ]]; then
            if [[ "$allow_insecure_password" == "true" ]]; then sz+=("-p${password}"); else sz+=("-p"); fi
          fi''',
    "7z extraction password",
)

zip_verify = '''          zip)
            if ! unzip -t "$output_file" >/dev/null 2>&1; then
              verify_ok=false
            fi
            ;;
          7z)
            if ! 7z t "$output_file" >/dev/null 2>&1; then
              verify_ok=false
            fi
            ;;
'''
zip_verify_safe = '''          zip)
            local -a verify_zip=(unzip -t)
            [[ "$password_requested" == "true" && "$allow_insecure_password" == "true" ]] && verify_zip+=(-P "$password")
            if ! "${verify_zip[@]}" "$output_file" >/dev/null 2>&1; then
              verify_ok=false
            fi
            ;;
          7z)
            local -a verify_7z=(7z t -bb0 -bd)
            if [[ "$password_requested" == "true" ]]; then
              if [[ "$allow_insecure_password" == "true" ]]; then verify_7z+=("-p${password}"); else verify_7z+=("-p"); fi
            fi
            if ! "${verify_7z[@]}" "$output_file" >/dev/null 2>&1; then
              verify_ok=false
            fi
            ;;
'''
text = replace_once(text, zip_verify, zip_verify_safe, "password verification")

text = replace_once(
    text,
    "      7z)   7z l -ba \"$f\" | awk '{print $NF}' ;;",
    "      7z)   7z l -slt -- \"$f\" | awk 'BEGIN { seen=0 } /^Path = / { if (seen++ == 0) next; sub(/^Path = /, \"\"); print }' ;;",
    "7z member listing",
)

extract_marker = "  extract_tar() {\n"
member_validation = r'''  archive_member_is_safe() {
    local member="$1"
    member="${member//$'\r'/}"
    member="${member//\\//}"
    while [[ "$member" == ./* ]]; do member="${member#./}"; done
    [[ -z "$member" || "$member" == "." ]] && return 0
    case "$member" in
      /*|..|../*|*/..|*/../*|[A-Za-z]:/*) return 1 ;;
    esac
    return 0
  }

  validate_archive_members() (
    local f="$1" fmt="$2" listing_file item
    listing_file=$(mktemp -t arx-members.XXXXXXXXXX) || return 1
    trap 'rm -f -- "$listing_file"' EXIT INT TERM
    if ! list_archive_contents "$f" "$fmt" > "$listing_file"; then
      log_error "Cannot inspect archive members safely: $f"
      return 1
    fi
    while IFS= read -r item || [[ -n "$item" ]]; do
      [[ -z "$item" ]] && continue
      if ! archive_member_is_safe "$item"; then
        log_error "Unsafe archive member path rejected: $item"
        return 1
      fi
    done < "$listing_file"
  )

'''
text = replace_once(text, extract_marker, member_validation + extract_marker, "archive member validation")

extraction_validation = '''      fmt=$(detect_format "$f")
      [[ "$fmt" == "unknown" ]] && { log_error "Unknown: $f"; exit_code=1; continue; }
      if [[ "$dry_run" == "true" ]]; then printf "%b\\n" "${DIM}[DRY-RUN]${NC} Extracting $f"; continue; fi
'''
extraction_validation_safe = '''      fmt=$(detect_format "$f")
      [[ "$fmt" == "unknown" ]] && { log_error "Unknown: $f"; exit_code=1; continue; }
      validate_archive_members "$f" "$fmt" || { exit_code=1; continue; }
      if [[ "$dry_run" == "true" ]]; then printf "%b\\n" "${DIM}[DRY-RUN]${NC} Extracting $f"; continue; fi
'''
text = replace_once(text, extraction_validation, extraction_validation_safe, "extraction validation call")

split_start = text.index("      # Split Archives\n")
split_end = text.index("      # Archive Verification\n", split_start)
text = text[:split_start] + text[split_end:]

split_after_verify = '''      # Verify the complete archive before splitting it into transport chunks.
      if [[ -n "${split_size:-}" ]]; then
        log_info "Splitting archive into ${split_size} chunks..."
        if split -b "$split_size" -d "$output_file" "${output_file}."; then
          rm -f -- "$output_file"
          log_success "Archive split into parts: ${output_file}.00, ${output_file}.01, ..."
        else
          log_error "Failed to split archive"
          return 1
        fi
      fi

'''
text = replace_once(
    text,
    '      if [[ "$delete_after" == "true" ]]; then\n',
    split_after_verify + '      if [[ "$delete_after" == "true" ]]; then\n',
    "move split after verification",
)

text = replace_once(
    text,
    '    local tmp_dir; tmp_dir=$(mktemp -d -t arx_convert.XXXXXXXXXX)\n',
    '''    local tmp_dir; tmp_dir=$(mktemp -d -t arx_convert.XXXXXXXXXX)
    trap 'rm -rf -- "$tmp_dir"' RETURN
    trap 'rm -rf -- "$tmp_dir"; log_error "Conversion aborted"; return 130' INT TERM
''',
    "conversion cleanup",
)

path.write_text(text)

test_path = Path("tests/test_arx.sh")
tests = test_path.read_text()
tests = replace_once(
    tests,
    '''# T3 interactive password works without --allow-insecure-password
printf 'secret\\n' | arx -c zip -p -n sec data/ >/dev/null 2>&1; ok $? "zip password via -p (no insecure flag)"
''',
    '''# T3 secure password prompts require a TTY; automation must opt into argv exposure
printf 'secret\\n' | arx -c zip -p -n sec data/ >/dev/null 2>&1; bad $? "password prompt rejects non-interactive input"
printf 'secret\\n' | arx -c zip -p --allow-insecure-password -n sec data/ >/dev/null 2>&1; ok $? "explicit insecure password mode works"
''',
    "password smoke test",
)

summary = '''echo ""
echo "PASS=$pass FAIL=$fail"
[[ $fail -eq 0 ]]
'''
additions = '''# T7 split-size validation and verify-before-split
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
'''
tests = replace_once(tests, summary, additions, "append smoke tests")
test_path.write_text(tests)
