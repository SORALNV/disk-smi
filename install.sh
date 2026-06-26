#!/bin/sh
set -eu

repo="SORALNV/disk-smi"
version="${DISK_SMI_VERSION:-latest}"
install_dir="${DISK_SMI_INSTALL_DIR:-$HOME/.local/bin}"
update_profile="${DISK_SMI_UPDATE_PROFILE:-1}"

log() {
	printf '%s\n' "$*"
}

fail() {
	printf 'disk-smi install: %s\n' "$*" >&2
	exit 1
}

case "$(uname -s)" in
	Darwin)
		os="darwin"
		;;
	*)
		fail "macOS only for now"
		;;
esac

case "$(uname -m)" in
	arm64|aarch64)
		arch="arm64"
		;;
	x86_64|amd64)
		arch="amd64"
		;;
	*)
		fail "unsupported architecture: $(uname -m)"
		;;
esac

asset="disk-smi-${os}-${arch}"
if [ "$version" = "latest" ]; then
	url="https://github.com/${repo}/releases/latest/download/${asset}"
else
	url="https://github.com/${repo}/releases/download/${version}/${asset}"
fi

tmp="${TMPDIR:-/tmp}/disk-smi-install.$$"
trap 'rm -f "$tmp"' EXIT INT TERM

mkdir -p "$install_dir"

log "Downloading ${asset} from ${repo} ${version}..."
if command -v curl >/dev/null 2>&1; then
	curl -fsSL -o "$tmp" "$url"
elif command -v wget >/dev/null 2>&1; then
	wget -q -O "$tmp" "$url"
else
	fail "curl or wget is required"
fi

chmod +x "$tmp"
mv "$tmp" "$install_dir/disk-smi"
xattr -d com.apple.quarantine "$install_dir/disk-smi" >/dev/null 2>&1 || true

case ":$PATH:" in
	*":$install_dir:"*)
		path_ready=1
		;;
	*)
		path_ready=0
		;;
esac

if [ "$path_ready" -eq 0 ] && [ "$update_profile" != "0" ]; then
	profile=""
	shell_name="$(basename "${SHELL:-}")"
	case "$shell_name" in
		zsh)
			profile="$HOME/.zshrc"
			;;
		bash)
			profile="$HOME/.bashrc"
			;;
	esac
	if [ -n "$profile" ]; then
		mkdir -p "$(dirname "$profile")"
		touch "$profile"
		if ! grep -F "$install_dir" "$profile" >/dev/null 2>&1; then
			{
				printf '\n'
				printf '# disk-smi\n'
				printf 'export PATH="%s:$PATH"\n' "$install_dir"
			} >> "$profile"
			log "Added ${install_dir} to ${profile}."
		fi
	fi
fi

log "Installed: ${install_dir}/disk-smi"
"$install_dir/disk-smi" --version

found="$(command -v disk-smi 2>/dev/null || true)"
if [ "$found" = "$install_dir/disk-smi" ]; then
	log "Ready: disk-smi -jp"
elif [ -n "$found" ]; then
	log "Note: 'disk-smi' currently resolves to ${found}."
	log "Installed this copy at ${install_dir}/disk-smi."
elif [ "$path_ready" -eq 0 ]; then
	log "Open a new terminal, or run:"
	log "  export PATH=\"${install_dir}:\$PATH\""
	log "Then run:"
	log "  disk-smi -jp"
else
	log "Run:"
	log "  disk-smi -jp"
fi
