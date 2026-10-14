#!/usr/bin/env bash
# ==============================================================================
# 🌼 Daisy Code Editor — Universal One-Line Installer
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/raillen/daisy-code-editor/main/scripts/install.sh | bash
#
# Non-interactive usage:
#   FORMAT=appimage VERSION=v0.2.0 bash install.sh
#   FORMAT=arch bash install.sh
#   FORMAT=deb bash install.sh
#   FORMAT=rpm bash install.sh
#   FORMAT=flatpak bash install.sh
# ==============================================================================
set -euo pipefail

REPO="raillen/daisy-code-editor"
DEFAULT_VERSION="v0.2.0"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
NO_PATH_MOD="${NO_PATH_MOD:-0}"
TMP_DIR=""

cleanup() {
    if [ -n "${TMP_DIR:-}" ] && [ -d "${TMP_DIR:-}" ]; then
        rm -rf "$TMP_DIR"
    fi
}
trap cleanup EXIT

# Formatting
BOLD="\033[1m"
GREEN="\033[32m"
BLUE="\033[34m"
YELLOW="\033[33m"
CYAN="\033[36m"
RED="\033[31m"
RESET="\033[0m"

log_info() {
    printf "${BLUE}==>${RESET} ${BOLD}%b${RESET}\n" "$1"
}

log_success() {
    printf "${GREEN}==>${RESET} ${BOLD}%b${RESET}\n" "$1"
}

log_warn() {
    printf "${YELLOW}==> WARNING:${RESET} %b\n" "$1"
}

log_error() {
    printf "${RED}==> ERROR:${RESET} %b\n" "$1" >&2
}

# Prompt helper that works even when piped into bash via `curl | bash`
read_input() {
    local prompt="$1"
    local var_name="$2"
    local default_val="${3:-}"
    local response=""

    if [ -t 0 ]; then
        printf "%b" "$prompt"
        read -r response
    elif [ -r /dev/tty ]; then
        printf "%b" "$prompt" > /dev/tty
        read -r response < /dev/tty
    else
        response="$default_val"
    fi

    if [ -z "$response" ]; then
        response="$default_val"
    fi

    printf -v "$var_name" "%s" "$response"
}

detect_distro() {
    if [ -f /etc/os-release ]; then
        # shellcheck disable=SC1091
        . /etc/os-release
        case "${ID:-}" in
            arch|manjaro|endeavouros|garuda|artix)
                echo "arch"
                return
                ;;
            debian|ubuntu|pop|mint|elementary|kali|raspbian)
                echo "deb"
                return
                ;;
            fedora|rhel|centos|rocky|almalinux|nobara)
                echo "rpm"
                return
                ;;
        esac

        case "${ID_LIKE:-}" in
            *arch*)   echo "arch"; return ;;
            *debian*|*ubuntu*) echo "deb"; return ;;
            *fedora*|*rhel*) echo "rpm"; return ;;
        esac
    fi

    if command -v pacman >/dev/null 2>&1; then
        echo "arch"
    elif command -v apt-get >/dev/null 2>&1 || command -v dpkg >/dev/null 2>&1; then
        echo "deb"
    elif command -v dnf >/dev/null 2>&1 || command -v rpm >/dev/null 2>&1; then
        echo "rpm"
    else
        echo "appimage"
    fi
}

get_available_versions() {
    local releases=""
    releases=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases" 2>/dev/null | grep '"tag_name":' | head -n 5 | sed -E 's/.*"([^"]+)".*/\1/' || true)
    if [ -n "$releases" ]; then
        echo "$releases"
    else
        echo "$DEFAULT_VERSION"
    fi
}

configure_path() {
    local target_dir="$1"
    case ":$PATH:" in
        *":$target_dir:"*) return 0 ;;
    esac

    if [ "$NO_PATH_MOD" = "1" ]; then
        log_warn "PATH modification skipped by request. Please add '$target_dir' to your PATH manually."
        return 0
    fi

    log_info "Adding '$target_dir' to shell PATH..."
    local export_line="export PATH=\"$target_dir:\$PATH\""
    local modified_any=0

    # Bash
    if [ -f "$HOME/.bashrc" ] && ! grep -qs "$export_line" "$HOME/.bashrc"; then
        printf "\n# Daisy Code Editor\n%s\n" "$export_line" >> "$HOME/.bashrc"
        modified_any=1
    fi

    # Zsh
    if [ -f "$HOME/.zshrc" ] && ! grep -qs "$export_line" "$HOME/.zshrc"; then
        printf "\n# Daisy Code Editor\n%s\n" "$export_line" >> "$HOME/.zshrc"
        modified_any=1
    fi

    # Profile
    if [ -f "$HOME/.profile" ] && ! grep -qs "$export_line" "$HOME/.profile"; then
        printf "\n# Daisy Code Editor\n%s\n" "$export_line" >> "$HOME/.profile"
        modified_any=1
    fi

    # Fish shell
    if [ -d "$HOME/.config/fish" ]; then
        local fish_config="$HOME/.config/fish/config.fish"
        mkdir -p "$HOME/.config/fish"
        if [ ! -f "$fish_config" ] || ! grep -qs "fish_add_path" "$fish_config"; then
            printf "\n# Daisy Code Editor\nfish_add_path ~/.local/bin\n" >> "$fish_config"
            modified_any=1
        fi
    fi

    if [ "$modified_any" = "1" ]; then
        log_success "Shell PATH updated. Restart your terminal or run: export PATH=\"$target_dir:\$PATH\""
    fi
}

install_desktop_integration() {
    local bin_path="$1"
    local apps_dir="$HOME/.local/share/applications"
    local icons_dir="$HOME/.local/share/icons/hicolor/scalable/apps"

    mkdir -p "$apps_dir" "$icons_dir"

    # Only retire a legacy Code Editor launcher, never a File Manager launcher.
    if [ -f "$apps_dir/daisy.desktop" ] && grep -Fxq "Name=Daisy Code Editor" "$apps_dir/daisy.desktop"; then
        rm -f "$apps_dir/daisy.desktop"
    fi

    cat << EOF > "$apps_dir/daisy-code.desktop"
[Desktop Entry]
Type=Application
Name=Daisy Code Editor
GenericName=Text Editor
Comment=Modern terminal code editor and mini-IDE built with OpenTUI + Solid
Exec=$bin_path %F
Icon=daisy-code
Terminal=true
Categories=Development;TextEditor;IDE;
Keywords=editor;code;terminal;tui;ide;daisy;
EOF

    cat << 'EOF' > "$icons_dir/daisy-code.svg"
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" width="128" height="128">
  <circle cx="64" cy="64" r="60" fill="#181825"/>
  <circle cx="64" cy="28" r="16" fill="#f5e0dc" opacity="0.9"/>
  <circle cx="64" cy="100" r="16" fill="#f5e0dc" opacity="0.9"/>
  <circle cx="28" cy="64" r="16" fill="#f5e0dc" opacity="0.9"/>
  <circle cx="100" cy="64" r="16" fill="#f5e0dc" opacity="0.9"/>
  <circle cx="38" cy="38" r="16" fill="#cdd6f4" opacity="0.9"/>
  <circle cx="90" cy="38" r="16" fill="#cdd6f4" opacity="0.9"/>
  <circle cx="38" cy="90" r="16" fill="#cdd6f4" opacity="0.9"/>
  <circle cx="90" cy="90" r="16" fill="#cdd6f4" opacity="0.9"/>
  <circle cx="64" cy="64" r="20" fill="#f9e2af"/>
  <circle cx="64" cy="64" r="14" fill="#fab387"/>
</svg>
EOF

    if command -v update-desktop-database >/dev/null 2>&1; then
        update-desktop-database "$apps_dir" >/dev/null 2>&1 || true
    fi
}


# Refuse native packages built with shared /usr/bin/daisy and daisy.desktop paths.
# v0.2.0 releases are not yet rebuilt with the isolated executable identity.
verify_native_package() {
    local format="$1" package="$2" contents=""
    case "$format" in
        arch) command -v tar >/dev/null 2>&1 || return 1
              contents=$(tar --zstd -tf "$package") || return 1 ;;
        deb)  command -v dpkg-deb >/dev/null 2>&1 || return 1
              contents=$(dpkg-deb --contents "$package" | awk '{ print $NF }') || return 1 ;;
        rpm)  command -v rpm >/dev/null 2>&1 || return 1
              contents=$(rpm -qpl "$package") || return 1 ;;
    esac
    if printf '%s\n' "$contents" | grep -Eq '(^|/)(daisy|daisy\.desktop|daisy\.svg|daisy\.png)    printf "\n"
    printf "  ${BOLD}${YELLOW}╭────────────────────────────────────────────────────────────╮${RESET}\n"
    printf "  ${BOLD}${YELLOW}│${RESET}               ${BOLD}🌼 Daisy Code Editor Installer${RESET}               ${BOLD}${YELLOW}│${RESET}\n"
    printf "  ${BOLD}${YELLOW}│${RESET}   ${CYAN}Modern Terminal Code Editor · OpenTUI + Solid (Bun)${RESET}     ${BOLD}${YELLOW}│${RESET}\n"
    printf "  ${BOLD}${YELLOW}╰────────────────────────────────────────────────────────────╯${RESET}\n\n"

    TMP_DIR=$(mktemp -d)
    local detected
    detected=$(detect_distro)

    # 1. Resolve Version
    local target_version="${VERSION:-}"
    if [ -z "$target_version" ]; then
        log_info "Fetching available releases..."
        local versions_list
        versions_list=$(get_available_versions)
        local latest_version
        latest_version=$(echo "$versions_list" | head -n 1)

        printf "\n  ${BOLD}Available versions:${RESET}\n"
        local i=1
        while IFS= read -r ver; do
            if [ -n "$ver" ]; then
                printf "    ${CYAN}[%d]${RESET} %s" "$i" "$ver"
                if [ "$ver" = "$latest_version" ]; then
                    printf " ${GREEN}(latest)${RESET}\n"
                else
                    printf "\n"
                fi
                ((i++))
            fi
        done <<< "$versions_list"

        local ver_choice=""
        read_input "\n  Select version to install [default: 1]: " ver_choice "1"
        
        # Check if choice is a number
        if [[ "$ver_choice" =~ ^[0-9]+$ ]]; then
            local selected_ver
            selected_ver=$(echo "$versions_list" | sed -n "${ver_choice}p")
            target_version="${selected_ver:-$latest_version}"
        elif [ -n "$ver_choice" ]; then
            target_version="$ver_choice"
        else
            target_version="$latest_version"
        fi
    fi

    # Ensure leading 'v'
    if [[ ! "$target_version" =~ ^v ]]; then
        target_version="v${target_version}"
    fi
    local raw_version="${target_version#v}"

    log_success "Selected release version: ${BOLD}${target_version}${RESET}"

    # 2. Resolve Installation Format
    local format="${FORMAT:-}"
    if [ -z "$format" ]; then
        printf "\n  ${BOLD}Choose installation format / package:${RESET}\n"
        printf "    ${CYAN}[1]${RESET} Recommended: isolated AppImage (no root or naming conflicts)\n"
        printf "    ${CYAN}[2]${RESET} AppImage (Universal standalone executable in ~/.local/bin)\n"
        printf "    ${CYAN}[3]${RESET} Arch Linux (.pkg.tar.zst via pacman)\n"
        printf "    ${CYAN}[4]${RESET} Debian / Ubuntu (.deb via apt/dpkg)\n"
        printf "    ${CYAN}[5]${RESET} Fedora / RHEL (.rpm via dnf/rpm)\n"
        printf "    ${CYAN}[6]${RESET} Flatpak (Universal flatpak bundle)\n"
        printf "    ${CYAN}[7]${RESET} Standalone Binary (Extracted directly to ~/.local/bin)\n"

        local fmt_choice=""
        read_input "\n  Select format [default: 1]: " fmt_choice "1"

        case "$fmt_choice" in
            1) format="appimage" ;;
            2) format="appimage" ;;
            3) format="arch" ;;
            4) format="deb" ;;
            5) format="rpm" ;;
            6) format="flatpak" ;;
            7) format="binary" ;;
            *) format="appimage" ;;
        esac
    fi

    format=$(echo "$format" | tr '[:upper:]' '[:lower:]')
    log_info "Target format: ${BOLD}${format^^}${RESET}"

    if [ "$(uname -s)" != "Linux" ] || [ "$(uname -m)" != "x86_64" ]; then
        log_error "Published assets only support Linux x86_64; detected $(uname -s) $(uname -m)."
        exit 1
    fi

    local base_url="https://github.com/${REPO}/releases/download/${target_version}"

    # Execute installation according to chosen format
    case "$format" in
        appimage)
            local file="Daisy-${raw_version}-x86_64.AppImage"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/daisy.AppImage"
            
            mkdir -p "$INSTALL_DIR"
            install -m 755 "$TMP_DIR/daisy.AppImage" "$INSTALL_DIR/daisy-code"
            verify_elf "$INSTALL_DIR/daisy-code"
            configure_path "$INSTALL_DIR"
            install_desktop_integration "$INSTALL_DIR/daisy-code"
            ;;


        arch)
            local file="daisy-code-editor-${raw_version}-1-x86_64.pkg.tar.zst"
            log_info "Downloading $file..."
            curl -fSL "${base_url}/$file" -o "$TMP_DIR/$file"
            verify_native_package arch "$TMP_DIR/$file"
            command -v pacman >/dev/null 2>&1 || { log_error "pacman required. Choose AppImage instead."; exit 1; }
            if [ "$(id -u)" -eq 0 ]; then pacman -U --noconfirm "$TMP_DIR/$file"
            else command -v sudo >/dev/null 2>&1 || { log_error "sudo required; choose AppImage."; exit 1; }
                 sudo pacman -U --noconfirm "$TMP_DIR/$file"; fi
            verify_native_install
            ;;
        deb)
            local file="daisy_${raw_version}_amd64.deb"
            log_info "Downloading $file..."
            curl -fSL "${base_url}/$file" -o "$TMP_DIR/$file"
            verify_native_package deb "$TMP_DIR/$file"
            command -v dpkg >/dev/null 2>&1 || { log_error "dpkg required. Choose AppImage instead."; exit 1; }
            if [ "$(id -u)" -eq 0 ]; then dpkg -i "$TMP_DIR/$file"
            else command -v sudo >/dev/null 2>&1 || { log_error "sudo required; choose AppImage."; exit 1; }
                 sudo dpkg -i "$TMP_DIR/$file"; fi
            verify_native_install
            ;;
        rpm)
            local file="daisy-code-editor-${raw_version}-1.x86_64.rpm"
            log_info "Downloading $file..."
            curl -fSL "${base_url}/$file" -o "$TMP_DIR/$file"
            verify_native_package rpm "$TMP_DIR/$file"
            if [ "$(id -u)" -eq 0 ]; then
                if command -v dnf >/dev/null 2>&1; then dnf install -y "$TMP_DIR/$file"
                else rpm -Uvh "$TMP_DIR/$file"; fi
            else
                command -v sudo >/dev/null 2>&1 || { log_error "sudo required; choose AppImage."; exit 1; }
                if command -v dnf >/dev/null 2>&1; then sudo dnf install -y "$TMP_DIR/$file"
                else sudo rpm -Uvh "$TMP_DIR/$file"; fi
            fi
            verify_native_install
            ;;

        flatpak)
            local file="daisy-${raw_version}-x86_64.flatpak"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/$file"

            if command -v flatpak >/dev/null 2>&1; then
                log_info "Installing Flatpak bundle..."
                flatpak install --user --bundle -y "$TMP_DIR/$file"
                flatpak info --user io.github.raillen.daisy >/dev/null || { log_error "Flatpak app ID was not installed."; exit 1; }
                log_success "Flatpak installed. Launch via flatpak run io.github.raillen.daisy"
            else
                log_error "Flatpak is not installed on this system."
                exit 1
            fi
            ;;

        binary)
            local file="Daisy-${raw_version}-x86_64.AppImage"
            log_info "Downloading ${file} to extract standalone binary..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/daisy.AppImage"
            chmod +x "$TMP_DIR/daisy.AppImage"

            mkdir -p "$TMP_DIR/extracted"
            (cd "$TMP_DIR/extracted" && "$TMP_DIR/daisy.AppImage" --appimage-extract >/dev/null 2>&1)
            
            mkdir -p "$INSTALL_DIR"
            cp "$TMP_DIR/extracted/squashfs-root/usr/bin/daisy" "$INSTALL_DIR/daisy-code"
            chmod 755 "$INSTALL_DIR/daisy-code"
            verify_elf "$INSTALL_DIR/daisy-code"

            configure_path "$INSTALL_DIR"
            install_desktop_integration "$INSTALL_DIR/daisy-code"
            ;;

        *)
            log_error "Unknown format: $format. Supported: appimage, arch, deb, rpm, flatpak, binary."
            exit 1
            ;;
    esac

    printf "\n"
    log_success "Daisy Code Editor (${target_version}) installed and verified! 🌼"

    printf "\n"
    printf "  ${BOLD}Getting Started:${RESET}\n"
    if [ "$format" = "appimage" ] || [ "$format" = "binary" ]; then
        printf "    ${CYAN}%s/daisy-code${RESET}  # Full path (works now)\n" "$INSTALL_DIR"
        printf "    ${YELLOW}If daisy-code is not found, restart your terminal or run:${RESET}\n"
        printf '    export PATH="%s:$PATH"\n' "$INSTALL_DIR"
    fi
    if [ "$format" = "flatpak" ]; then
        printf "    ${CYAN}flatpak run io.github.raillen.daisy${RESET}\n"
        return 0
    fi
    printf "    ${CYAN}daisy-code .${RESET}             # Open current folder in Daisy\n"
    printf "    ${CYAN}daisy-code path/to/file${RESET}  # Open a specific file\n"
    printf "    ${CYAN}daisy-code --help${RESET}        # View keyboard shortcuts & options\n\n"
}

main "$@"
; then
        log_error "Package contains legacy Daisy paths and could overwrite Daisy File Manager."
        log_error "No package was installed. Choose AppImage (option 2) until native packages are rebuilt."
        return 1
    fi
    if ! printf '%s\n' "$contents" | grep -Eq '(^|/)usr/bin/daisy-code    printf "\n"
    printf "  ${BOLD}${YELLOW}╭────────────────────────────────────────────────────────────╮${RESET}\n"
    printf "  ${BOLD}${YELLOW}│${RESET}               ${BOLD}🌼 Daisy Code Editor Installer${RESET}               ${BOLD}${YELLOW}│${RESET}\n"
    printf "  ${BOLD}${YELLOW}│${RESET}   ${CYAN}Modern Terminal Code Editor · OpenTUI + Solid (Bun)${RESET}     ${BOLD}${YELLOW}│${RESET}\n"
    printf "  ${BOLD}${YELLOW}╰────────────────────────────────────────────────────────────╯${RESET}\n\n"

    TMP_DIR=$(mktemp -d)
    local detected
    detected=$(detect_distro)

    # 1. Resolve Version
    local target_version="${VERSION:-}"
    if [ -z "$target_version" ]; then
        log_info "Fetching available releases..."
        local versions_list
        versions_list=$(get_available_versions)
        local latest_version
        latest_version=$(echo "$versions_list" | head -n 1)

        printf "\n  ${BOLD}Available versions:${RESET}\n"
        local i=1
        while IFS= read -r ver; do
            if [ -n "$ver" ]; then
                printf "    ${CYAN}[%d]${RESET} %s" "$i" "$ver"
                if [ "$ver" = "$latest_version" ]; then
                    printf " ${GREEN}(latest)${RESET}\n"
                else
                    printf "\n"
                fi
                ((i++))
            fi
        done <<< "$versions_list"

        local ver_choice=""
        read_input "\n  Select version to install [default: 1]: " ver_choice "1"
        
        # Check if choice is a number
        if [[ "$ver_choice" =~ ^[0-9]+$ ]]; then
            local selected_ver
            selected_ver=$(echo "$versions_list" | sed -n "${ver_choice}p")
            target_version="${selected_ver:-$latest_version}"
        elif [ -n "$ver_choice" ]; then
            target_version="$ver_choice"
        else
            target_version="$latest_version"
        fi
    fi

    # Ensure leading 'v'
    if [[ ! "$target_version" =~ ^v ]]; then
        target_version="v${target_version}"
    fi
    local raw_version="${target_version#v}"

    log_success "Selected release version: ${BOLD}${target_version}${RESET}"

    # 2. Resolve Installation Format
    local format="${FORMAT:-}"
    if [ -z "$format" ]; then
        printf "\n  ${BOLD}Choose installation format / package:${RESET}\n"
        printf "    ${CYAN}[1]${RESET} Recommended: isolated AppImage (no root or naming conflicts)\n"
        printf "    ${CYAN}[2]${RESET} AppImage (Universal standalone executable in ~/.local/bin)\n"
        printf "    ${CYAN}[3]${RESET} Arch Linux (.pkg.tar.zst via pacman)\n"
        printf "    ${CYAN}[4]${RESET} Debian / Ubuntu (.deb via apt/dpkg)\n"
        printf "    ${CYAN}[5]${RESET} Fedora / RHEL (.rpm via dnf/rpm)\n"
        printf "    ${CYAN}[6]${RESET} Flatpak (Universal flatpak bundle)\n"
        printf "    ${CYAN}[7]${RESET} Standalone Binary (Extracted directly to ~/.local/bin)\n"

        local fmt_choice=""
        read_input "\n  Select format [default: 1]: " fmt_choice "1"

        case "$fmt_choice" in
            1) format="appimage" ;;
            2) format="appimage" ;;
            3) format="arch" ;;
            4) format="deb" ;;
            5) format="rpm" ;;
            6) format="flatpak" ;;
            7) format="binary" ;;
            *) format="appimage" ;;
        esac
    fi

    format=$(echo "$format" | tr '[:upper:]' '[:lower:]')
    log_info "Target format: ${BOLD}${format^^}${RESET}"

    local base_url="https://github.com/${REPO}/releases/download/${target_version}"

    # Execute installation according to chosen format
    case "$format" in
        appimage)
            local file="Daisy-${raw_version}-x86_64.AppImage"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/daisy.AppImage"
            
            mkdir -p "$INSTALL_DIR"
            install -m 755 "$TMP_DIR/daisy.AppImage" "$INSTALL_DIR/daisy-code"
            configure_path "$INSTALL_DIR"
            install_desktop_integration "$INSTALL_DIR/daisy-code"
            ;;

        arch)
            local file="daisy-code-editor-${raw_version}-1-x86_64.pkg.tar.zst"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/$file"
            
            if command -v pacman >/dev/null 2>&1; then
                log_info "Installing via pacman..."
                if [ "$(id -u)" -eq 0 ]; then
                    pacman -U --noconfirm "$TMP_DIR/$file"
                elif command -v sudo >/dev/null 2>&1; then
                    sudo pacman -U --noconfirm "$TMP_DIR/$file"
                else
                    log_warn "No root or sudo available. Extracting package to ~/.local..."
                    tar --zstd -xf "$TMP_DIR/$file" -C "$HOME/.local"
                    configure_path "$INSTALL_DIR"
                fi
            else
                log_warn "pacman not found. Extracting package contents into ~/.local..."
                tar --zstd -xf "$TMP_DIR/$file" -C "$HOME/.local"
                configure_path "$INSTALL_DIR"
            fi
            ;;

        deb)
            local file="daisy_${raw_version}_amd64.deb"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/$file"

            if command -v apt-get >/dev/null 2>&1 || command -v dpkg >/dev/null 2>&1; then
                log_info "Installing Debian package..."
                if [ "$(id -u)" -eq 0 ]; then
                    apt-get install -y "$TMP_DIR/$file" || dpkg -i "$TMP_DIR/$file"
                elif command -v sudo >/dev/null 2>&1; then
                    sudo apt-get install -y "$TMP_DIR/$file" || sudo dpkg -i "$TMP_DIR/$file"
                else
                    log_warn "No root or sudo available. Extracting .deb data to ~/.local..."
                    ar p "$TMP_DIR/$file" data.tar.zst 2>/dev/null | tar --zstd -xf - -C "$HOME/.local" || \
                    ar p "$TMP_DIR/$file" data.tar.xz 2>/dev/null | tar -xJf - -C "$HOME/.local" || \
                    ar p "$TMP_DIR/$file" data.tar.gz 2>/dev/null | tar -xzf - -C "$HOME/.local"
                    configure_path "$INSTALL_DIR"
                fi
            else
                log_warn "apt/dpkg not found. Extracting .deb to ~/.local..."
                ar p "$TMP_DIR/$file" data.tar.zst 2>/dev/null | tar --zstd -xf - -C "$HOME/.local" || true
                configure_path "$INSTALL_DIR"
            fi
            ;;

        rpm)
            local file="daisy-code-editor-${raw_version}-1.x86_64.rpm"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/$file"

            if command -v dnf >/dev/null 2>&1 || command -v rpm >/dev/null 2>&1; then
                log_info "Installing RPM package..."
                if [ "$(id -u)" -eq 0 ]; then
                    dnf install -y "$TMP_DIR/$file" || rpm -Uvh "$TMP_DIR/$file"
                elif command -v sudo >/dev/null 2>&1; then
                    sudo dnf install -y "$TMP_DIR/$file" || sudo rpm -Uvh "$TMP_DIR/$file"
                else
                    log_warn "No root or sudo available. Extracting RPM to ~/.local..."
                    cd "$HOME/.local" && rpm2cpio "$TMP_DIR/$file" | cpio -idmv
                    configure_path "$INSTALL_DIR"
                fi
            else
                log_warn "dnf/rpm not found. Extracting RPM to ~/.local..."
                cd "$HOME/.local" && rpm2cpio "$TMP_DIR/$file" | cpio -idmv
                configure_path "$INSTALL_DIR"
            fi
            ;;

        flatpak)
            local file="daisy-${raw_version}-x86_64.flatpak"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/$file"

            if command -v flatpak >/dev/null 2>&1; then
                log_info "Installing Flatpak bundle..."
                flatpak install --user --bundle -y "$TMP_DIR/$file"
                log_success "Daisy installed as Flatpak: run with 'flatpak run io.github.raillen.daisy'"
            else
                log_error "Flatpak is not installed on this system."
                exit 1
            fi
            ;;

        binary)
            local file="Daisy-${raw_version}-x86_64.AppImage"
            log_info "Downloading ${file} to extract standalone binary..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/daisy.AppImage"
            chmod +x "$TMP_DIR/daisy.AppImage"

            mkdir -p "$TMP_DIR/extracted"
            (cd "$TMP_DIR/extracted" && "$TMP_DIR/daisy.AppImage" --appimage-extract >/dev/null 2>&1)
            
            mkdir -p "$INSTALL_DIR"
            cp "$TMP_DIR/extracted/squashfs-root/usr/bin/daisy" "$INSTALL_DIR/daisy-code"
            chmod 755 "$INSTALL_DIR/daisy-code"

            configure_path "$INSTALL_DIR"
            install_desktop_integration "$INSTALL_DIR/daisy-code"
            ;;

        *)
            log_error "Unknown format: $format. Supported: appimage, arch, deb, rpm, flatpak, binary."
            exit 1
            ;;
    esac

    printf "\n"
    log_success "Daisy Code Editor (${target_version}) successfully installed! 🌼"

    printf "\n"
    printf "  ${BOLD}Getting Started:${RESET}\n"
    printf "    ${CYAN}daisy-code .${RESET}             # Open current folder in Daisy\n"
    printf "    ${CYAN}daisy-code path/to/file${RESET}  # Open a specific file\n"
    printf "    ${CYAN}daisy-code --help${RESET}        # View keyboard shortcuts & options\n\n"
}

main "$@"
; then
        log_error "Package does not contain /usr/bin/daisy-code. Installation refused."
        return 1
    fi
}

verify_elf() {
    local target="$1" magic
    [ -s "$target" ] && [ -x "$target" ] || { log_error "Installed file is missing or non-executable: $target"; return 1; }
    magic=$(od -An -tx1 -N4 "$target" | tr -d ' \n')
    [ "$magic" = "7f454c46" ] || { log_error "Artifact is not a Linux executable: $target"; return 1; }
}

verify_native_install() {
    [ -x /usr/bin/daisy-code ] || [ -x /usr/local/bin/daisy-code ] || {
        log_error "Package manager exited successfully, but /usr/bin/daisy-code is missing."
        return 1
    }
}

main() {
    printf "\n"
    printf "  ${BOLD}${YELLOW}╭────────────────────────────────────────────────────────────╮${RESET}\n"
    printf "  ${BOLD}${YELLOW}│${RESET}               ${BOLD}🌼 Daisy Code Editor Installer${RESET}               ${BOLD}${YELLOW}│${RESET}\n"
    printf "  ${BOLD}${YELLOW}│${RESET}   ${CYAN}Modern Terminal Code Editor · OpenTUI + Solid (Bun)${RESET}     ${BOLD}${YELLOW}│${RESET}\n"
    printf "  ${BOLD}${YELLOW}╰────────────────────────────────────────────────────────────╯${RESET}\n\n"

    TMP_DIR=$(mktemp -d)
    local detected
    detected=$(detect_distro)

    # 1. Resolve Version
    local target_version="${VERSION:-}"
    if [ -z "$target_version" ]; then
        log_info "Fetching available releases..."
        local versions_list
        versions_list=$(get_available_versions)
        local latest_version
        latest_version=$(echo "$versions_list" | head -n 1)

        printf "\n  ${BOLD}Available versions:${RESET}\n"
        local i=1
        while IFS= read -r ver; do
            if [ -n "$ver" ]; then
                printf "    ${CYAN}[%d]${RESET} %s" "$i" "$ver"
                if [ "$ver" = "$latest_version" ]; then
                    printf " ${GREEN}(latest)${RESET}\n"
                else
                    printf "\n"
                fi
                ((i++))
            fi
        done <<< "$versions_list"

        local ver_choice=""
        read_input "\n  Select version to install [default: 1]: " ver_choice "1"
        
        # Check if choice is a number
        if [[ "$ver_choice" =~ ^[0-9]+$ ]]; then
            local selected_ver
            selected_ver=$(echo "$versions_list" | sed -n "${ver_choice}p")
            target_version="${selected_ver:-$latest_version}"
        elif [ -n "$ver_choice" ]; then
            target_version="$ver_choice"
        else
            target_version="$latest_version"
        fi
    fi

    # Ensure leading 'v'
    if [[ ! "$target_version" =~ ^v ]]; then
        target_version="v${target_version}"
    fi
    local raw_version="${target_version#v}"

    log_success "Selected release version: ${BOLD}${target_version}${RESET}"

    # 2. Resolve Installation Format
    local format="${FORMAT:-}"
    if [ -z "$format" ]; then
        printf "\n  ${BOLD}Choose installation format / package:${RESET}\n"
        printf "    ${CYAN}[1]${RESET} Recommended: isolated AppImage (no root or naming conflicts)\n"
        printf "    ${CYAN}[2]${RESET} AppImage (Universal standalone executable in ~/.local/bin)\n"
        printf "    ${CYAN}[3]${RESET} Arch Linux (.pkg.tar.zst via pacman)\n"
        printf "    ${CYAN}[4]${RESET} Debian / Ubuntu (.deb via apt/dpkg)\n"
        printf "    ${CYAN}[5]${RESET} Fedora / RHEL (.rpm via dnf/rpm)\n"
        printf "    ${CYAN}[6]${RESET} Flatpak (Universal flatpak bundle)\n"
        printf "    ${CYAN}[7]${RESET} Standalone Binary (Extracted directly to ~/.local/bin)\n"

        local fmt_choice=""
        read_input "\n  Select format [default: 1]: " fmt_choice "1"

        case "$fmt_choice" in
            1) format="appimage" ;;
            2) format="appimage" ;;
            3) format="arch" ;;
            4) format="deb" ;;
            5) format="rpm" ;;
            6) format="flatpak" ;;
            7) format="binary" ;;
            *) format="appimage" ;;
        esac
    fi

    format=$(echo "$format" | tr '[:upper:]' '[:lower:]')
    log_info "Target format: ${BOLD}${format^^}${RESET}"

    local base_url="https://github.com/${REPO}/releases/download/${target_version}"

    # Execute installation according to chosen format
    case "$format" in
        appimage)
            local file="Daisy-${raw_version}-x86_64.AppImage"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/daisy.AppImage"
            
            mkdir -p "$INSTALL_DIR"
            install -m 755 "$TMP_DIR/daisy.AppImage" "$INSTALL_DIR/daisy-code"
            configure_path "$INSTALL_DIR"
            install_desktop_integration "$INSTALL_DIR/daisy-code"
            ;;

        arch)
            local file="daisy-code-editor-${raw_version}-1-x86_64.pkg.tar.zst"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/$file"
            
            if command -v pacman >/dev/null 2>&1; then
                log_info "Installing via pacman..."
                if [ "$(id -u)" -eq 0 ]; then
                    pacman -U --noconfirm "$TMP_DIR/$file"
                elif command -v sudo >/dev/null 2>&1; then
                    sudo pacman -U --noconfirm "$TMP_DIR/$file"
                else
                    log_warn "No root or sudo available. Extracting package to ~/.local..."
                    tar --zstd -xf "$TMP_DIR/$file" -C "$HOME/.local"
                    configure_path "$INSTALL_DIR"
                fi
            else
                log_warn "pacman not found. Extracting package contents into ~/.local..."
                tar --zstd -xf "$TMP_DIR/$file" -C "$HOME/.local"
                configure_path "$INSTALL_DIR"
            fi
            ;;

        deb)
            local file="daisy_${raw_version}_amd64.deb"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/$file"

            if command -v apt-get >/dev/null 2>&1 || command -v dpkg >/dev/null 2>&1; then
                log_info "Installing Debian package..."
                if [ "$(id -u)" -eq 0 ]; then
                    apt-get install -y "$TMP_DIR/$file" || dpkg -i "$TMP_DIR/$file"
                elif command -v sudo >/dev/null 2>&1; then
                    sudo apt-get install -y "$TMP_DIR/$file" || sudo dpkg -i "$TMP_DIR/$file"
                else
                    log_warn "No root or sudo available. Extracting .deb data to ~/.local..."
                    ar p "$TMP_DIR/$file" data.tar.zst 2>/dev/null | tar --zstd -xf - -C "$HOME/.local" || \
                    ar p "$TMP_DIR/$file" data.tar.xz 2>/dev/null | tar -xJf - -C "$HOME/.local" || \
                    ar p "$TMP_DIR/$file" data.tar.gz 2>/dev/null | tar -xzf - -C "$HOME/.local"
                    configure_path "$INSTALL_DIR"
                fi
            else
                log_warn "apt/dpkg not found. Extracting .deb to ~/.local..."
                ar p "$TMP_DIR/$file" data.tar.zst 2>/dev/null | tar --zstd -xf - -C "$HOME/.local" || true
                configure_path "$INSTALL_DIR"
            fi
            ;;

        rpm)
            local file="daisy-code-editor-${raw_version}-1.x86_64.rpm"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/$file"

            if command -v dnf >/dev/null 2>&1 || command -v rpm >/dev/null 2>&1; then
                log_info "Installing RPM package..."
                if [ "$(id -u)" -eq 0 ]; then
                    dnf install -y "$TMP_DIR/$file" || rpm -Uvh "$TMP_DIR/$file"
                elif command -v sudo >/dev/null 2>&1; then
                    sudo dnf install -y "$TMP_DIR/$file" || sudo rpm -Uvh "$TMP_DIR/$file"
                else
                    log_warn "No root or sudo available. Extracting RPM to ~/.local..."
                    cd "$HOME/.local" && rpm2cpio "$TMP_DIR/$file" | cpio -idmv
                    configure_path "$INSTALL_DIR"
                fi
            else
                log_warn "dnf/rpm not found. Extracting RPM to ~/.local..."
                cd "$HOME/.local" && rpm2cpio "$TMP_DIR/$file" | cpio -idmv
                configure_path "$INSTALL_DIR"
            fi
            ;;

        flatpak)
            local file="daisy-${raw_version}-x86_64.flatpak"
            log_info "Downloading ${file}..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/$file"

            if command -v flatpak >/dev/null 2>&1; then
                log_info "Installing Flatpak bundle..."
                flatpak install --user --bundle -y "$TMP_DIR/$file"
                log_success "Daisy installed as Flatpak: run with 'flatpak run io.github.raillen.daisy'"
            else
                log_error "Flatpak is not installed on this system."
                exit 1
            fi
            ;;

        binary)
            local file="Daisy-${raw_version}-x86_64.AppImage"
            log_info "Downloading ${file} to extract standalone binary..."
            curl -fSL "${base_url}/${file}" -o "$TMP_DIR/daisy.AppImage"
            chmod +x "$TMP_DIR/daisy.AppImage"

            mkdir -p "$TMP_DIR/extracted"
            (cd "$TMP_DIR/extracted" && "$TMP_DIR/daisy.AppImage" --appimage-extract >/dev/null 2>&1)
            
            mkdir -p "$INSTALL_DIR"
            cp "$TMP_DIR/extracted/squashfs-root/usr/bin/daisy" "$INSTALL_DIR/daisy-code"
            chmod 755 "$INSTALL_DIR/daisy-code"

            configure_path "$INSTALL_DIR"
            install_desktop_integration "$INSTALL_DIR/daisy-code"
            ;;

        *)
            log_error "Unknown format: $format. Supported: appimage, arch, deb, rpm, flatpak, binary."
            exit 1
            ;;
    esac

    printf "\n"
    log_success "Daisy Code Editor (${target_version}) successfully installed! 🌼"

    printf "\n"
    printf "  ${BOLD}Getting Started:${RESET}\n"
    printf "    ${CYAN}daisy-code .${RESET}             # Open current folder in Daisy\n"
    printf "    ${CYAN}daisy-code path/to/file${RESET}  # Open a specific file\n"
    printf "    ${CYAN}daisy-code --help${RESET}        # View keyboard shortcuts & options\n\n"
}

main "$@"
