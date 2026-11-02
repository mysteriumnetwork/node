#!/usr/bin/env bash
set -euo pipefail
root=$(mktemp -d); trap 'rm -rf "$root"' EXIT
mock="$root/mock-bin"; mkdir -p "$mock"
cat >"$mock/curl" <<'EOF'
#!/usr/bin/env bash
set -e
[[ "${MOCK_CURL_FAIL:-0}" == 1 ]] && exit 1
out=""; while (($#)); do [[ "$1" == "-o" ]] && { out=$2; shift 2; continue; }; shift; done
printf 'mock-public-key\n' >"$out"
EOF
cat >"$mock/gpg" <<'EOF'
#!/usr/bin/env bash
set -e
if [[ "$*" == *"--import "* ]]; then
 case "${MOCK_GPG_MODE:-valid}" in
  valid) printf 'pub:::::::::\nfpr:::::::::BC7F93152681BDE3356F4080A197201433627AB4:\n' ;;
  wrong) printf 'pub:::::::::\nfpr:::::::::AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA:\n' ;;
  corrupt|partial) printf 'pub:::::::::\nfpr:::::::::BC7F93152681BDE3356F4080A197201433627AB4:\n'; exit 1 ;;
  private) printf 'sec:::::::::\nssb:::::::::\n' ;;
  extra) printf 'pub:::::::::\nfpr:::::::::BC7F93152681BDE3356F4080A197201433627AB4:\npub:::::::::\nfpr:::::::::AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA:\n' ;;
  *) exit 1;;
 esac
else
 out=""; while (($#)); do [[ "$1" == "-o" ]] && { out=$2; shift 2; continue; }; shift; done
 printf 'dearmored-key\n' >"$out"
fi
EOF
cat >"$mock/install" <<'EOF'
#!/usr/bin/env bash
set -e
[[ "${MOCK_INSTALL_FAIL:-0}" == 1 ]] && exit 1
while (($# > 2)); do shift; done
cp "$1" "$2"; chmod 0644 "$2"
EOF
cat >"$mock/mv" <<'EOF'
#!/usr/bin/env bash
set -e
f=${MOCK_MV_COUNT_FILE:?}; n=0; [[ -f "$f" ]] && n=$(<"$f"); n=$((n+1)); printf '%s\n' "$n" >"$f"
[[ "${MOCK_MV_FAIL_ON:-0}" == "$n" ]] && exit 1
exec /bin/mv "$@"
EOF
cat >"$mock/apt" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${MOCK_APT_LOG:?}"; exit 1
EOF
chmod +x "$mock"/*
setup() {
 case_dir="$root/$1"; mkdir -p "$case_dir/keyrings" "$case_dir/sources"
 export PATH="$mock:$PATH" APT_KEYRING_DIR="$case_dir/keyrings" APT_SOURCE_DIR="$case_dir/sources" MYSTERIUM_INSTALLER_LIBRARY=1
 export MOCK_MV_COUNT_FILE="$case_dir/mv-count" MOCK_APT_LOG="$case_dir/apt.log"
 export MOCK_CURL_FAIL=0 MOCK_GPG_MODE=valid MOCK_INSTALL_FAIL=0 MOCK_MV_FAIL_ON=0; : >"$MOCK_APT_LOG"
}
run_case() {
 setup "$1"; case "$2" in snapshot) SNAPSHOT=true NETWORK=;; testnet3) SNAPSHOT= NETWORK=testnet3;; *) SNAPSHOT= NETWORK=;; esac; export SNAPSHOT NETWORK
 source "$PWD/install.sh"; configure_mysterium_repository
 [[ -s "$APT_KEYRING_DIR/mysterium.gpg" ]]
 grep -Fq "signed-by=$APT_KEYRING_DIR/mysterium.gpg" "$APT_SOURCE_DIR/mysterium.list"
 case "$2" in snapshot) grep -Fq node-dev/ubuntu "$APT_SOURCE_DIR/mysterium.list";; testnet3) grep -Fq node-testnet3/ubuntu "$APT_SOURCE_DIR/mysterium.list";; *) grep -Fq mysteriumnetwork/node/ubuntu "$APT_SOURCE_DIR/mysterium.list";; esac
}
run_case stable-valid stable; run_case snapshot-valid snapshot; run_case testnet3-valid testnet3
for mode in wrong corrupt partial private extra; do
 setup "reject-$mode"; export MOCK_GPG_MODE=$mode; source "$PWD/install.sh"
 if configure_mysterium_repository; then exit 1; fi
 [[ ! -e "$APT_KEYRING_DIR/mysterium.gpg" && ! -e "$APT_SOURCE_DIR/mysterium.list" ]]
done
setup curl-failure; export MOCK_CURL_FAIL=1; source "$PWD/install.sh"; if configure_mysterium_repository; then exit 1; fi
setup install-failure; export MOCK_INSTALL_FAIL=1; source "$PWD/install.sh"; if configure_mysterium_repository; then exit 1; fi
setup first-mv-failure; export MOCK_MV_FAIL_ON=1; source "$PWD/install.sh"; if configure_mysterium_repository; then exit 1; fi
setup second-mv-failure; export MOCK_MV_FAIL_ON=2; source "$PWD/install.sh"; if configure_mysterium_repository; then exit 1; fi
[[ -s "$APT_KEYRING_DIR/mysterium.gpg" && ! -e "$APT_SOURCE_DIR/mysterium.list" ]]
setup foreign-source; printf 'deb https://example.invalid/debian stable main\n' >"$APT_SOURCE_DIR/foreign.list"; source "$PWD/install.sh"; configure_mysterium_repository; grep -Fxq 'deb https://example.invalid/debian stable main' "$APT_SOURCE_DIR/foreign.list"
setup mixed-source; printf 'deb https://example.invalid/debian stable main\ndeb http://ppa.launchpad.net/mysteriumnetwork/node/ubuntu focal main\n' >"$APT_SOURCE_DIR/mysterium.list"; source "$PWD/install.sh"; configure_mysterium_repository; grep -Fxq 'deb https://example.invalid/debian stable main' "$APT_SOURCE_DIR/mysterium.list"; ! grep -Fq 'ppa.launchpad.net' "$APT_SOURCE_DIR/mysterium.list"
channel_switch() {
 setup "switch-$1-$2"
 printf '%s\n' "$3" >"$APT_SOURCE_DIR/mysterium.list"
 case "$2" in snapshot) SNAPSHOT=true NETWORK=;; testnet3) SNAPSHOT= NETWORK=testnet3;; *) SNAPSHOT= NETWORK=;; esac; export SNAPSHOT NETWORK
 source "$PWD/install.sh"; configure_mysterium_repository
 [[ "$(grep -Eic 'https?://(ppa\.mysterium\.network|ppa\.launchpad\.net)/mysteriumnetwork/node(-dev|-testnet3)?/ubuntu' "$APT_SOURCE_DIR/mysterium.list")" == 1 ]]
 grep -Fq "signed-by=$APT_KEYRING_DIR/mysterium.gpg" "$APT_SOURCE_DIR/mysterium.list"
 case "$2" in snapshot) grep -Fq node-dev/ubuntu "$APT_SOURCE_DIR/mysterium.list";; testnet3) grep -Fq node-testnet3/ubuntu "$APT_SOURCE_DIR/mysterium.list";; *) grep -Fq 'mysteriumnetwork/node/ubuntu' "$APT_SOURCE_DIR/mysterium.list";; esac
}
channel_switch stable snapshot 'deb https://ppa.mysterium.network/mysteriumnetwork/node/ubuntu focal main'
channel_switch snapshot testnet3 'deb https://ppa.mysterium.network/mysteriumnetwork/node-dev/ubuntu focal main'
channel_switch testnet3 stable 'deb https://ppa.mysterium.network/mysteriumnetwork/node-testnet3/ubuntu focal main'
setup all-channels; printf '%s\n%s\n%s\n' 'deb https://ppa.mysterium.network/mysteriumnetwork/node/ubuntu focal main' 'deb http://ppa.launchpad.net/mysteriumnetwork/node-dev/ubuntu focal main' 'deb https://ppa.mysterium.network/mysteriumnetwork/node-testnet3/ubuntu focal main' >"$APT_SOURCE_DIR/mysterium.list"; SNAPSHOT=true NETWORK=; export SNAPSHOT NETWORK; source "$PWD/install.sh"; configure_mysterium_repository; [[ "$(grep -Eic 'https?://(ppa\.mysterium\.network|ppa\.launchpad\.net)/mysteriumnetwork/node(-dev|-testnet3)?/ubuntu' "$APT_SOURCE_DIR/mysterium.list")" == 1 ]]; grep -Fq node-dev/ubuntu "$APT_SOURCE_DIR/mysterium.list"
setup foreign-launchpad; printf '%s\n%s\n' 'deb https://ppa.launchpad.net/other/project/ubuntu focal main' 'deb https://ppa.mysterium.network/mysteriumnetwork/node/ubuntu focal main' >"$APT_SOURCE_DIR/mysterium.list"; SNAPSHOT= NETWORK=testnet3; export SNAPSHOT NETWORK; source "$PWD/install.sh"; configure_mysterium_repository; grep -Fxq 'deb https://ppa.launchpad.net/other/project/ubuntu focal main' "$APT_SOURCE_DIR/mysterium.list"
setup no-existing-file; SNAPSHOT= NETWORK=; export SNAPSHOT NETWORK; source "$PWD/install.sh"; configure_mysterium_repository; grep -Fq 'mysteriumnetwork/node/ubuntu' "$APT_SOURCE_DIR/mysterium.list"
setup same-channel; printf 'deb https://ppa.mysterium.network/mysteriumnetwork/node/ubuntu focal main\n' >"$APT_SOURCE_DIR/mysterium.list"; SNAPSHOT= NETWORK=; export SNAPSHOT NETWORK; source "$PWD/install.sh"; configure_mysterium_repository; [[ "$(grep -Fc 'mysteriumnetwork/node/ubuntu' "$APT_SOURCE_DIR/mysterium.list")" == 1 ]]
setup no-apt-on-error; export MOCK_GPG_MODE=wrong; source "$PWD/install.sh"; if configure_mysterium_repository; then exit 1; fi; [[ ! -s "$MOCK_APT_LOG" ]]
echo 'install.sh isolated tests passed'
