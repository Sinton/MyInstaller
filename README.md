# my-pnpm-installer

A cross-platform terminal user interface (TUI) tool for managing and installing pnpm/npm global packages. Define your package list in a YAML configuration file and manage installations through an intuitive interactive interface.

## Features

- **Interactive TUI**: Clean, keyboard-driven interface built with Bubble Tea
- **Version Management**: View latest and installed versions side-by-side
- **Smart Status Detection**: Automatically detects if packages need updates
- **Real-time Installation Logs**: Watch installation progress in real-time
- **Cross-Platform**: Works on Windows, macOS, and Linux
- **Flexible Configuration**: Define packages with custom commands via YAML
- **Concurrent Version Checks**: Fast parallel version lookups

## Installation

### Prerequisites

- Go 1.21 or higher
- npm (v7+) or pnpm (v7+)
- Node.js (v14+)

### Build from Source

```bash
git clone https://github.com/Sinton/my-pnpm-installer.git
cd my-pnpm-installer
go build -o my-pnpm-installer
```

### Install

```bash
# Move to a directory in your PATH
sudo mv my-pnpm-installer /usr/local/bin/  # Unix/macOS
# or
move my-pnpm-installer.exe C:\Windows\System32\  # Windows
```

## Usage

### Quick Start

1. Create a `config.yaml` file in your current directory or `~/.config/pnpm-manager/`:

```yaml
packages:
  - name: typescript
    display_name: TypeScript
    install_command: npm install -g typescript@latest
    version_check_command: npm view typescript version
    local_version_command: tsc --version

  - name: pnpm
    display_name: pnpm Package Manager
    install_command: npm install -g pnpm@latest
    version_check_command: npm view pnpm version
    local_version_command: pnpm --version
```

2. Run the tool:

```bash
my-pnpm-installer
```

### Keyboard Controls

- **↑/↓** or **j/k**: Navigate package list
- **Enter**: Install or update selected package
- **r**: Refresh package information
- **q** or **Ctrl+C**: Quit

### Interface Layout

```
┌─────────────────────────┬─────────────────────────┐
│ Package List            │ Package Details         │
│                         │                         │
│ > TypeScript            │ Package: TypeScript     │
│   pnpm Package Manager  │ Latest: 5.3.3           │
│   Qwen Code CLI         │ Installed: 5.2.2        │
│                         │ Status: Update Available│
└─────────────────────────┴─────────────────────────┘
[Enter] Install/Update  [r] Refresh  [q] Quit
```

## Configuration

### config.yaml Format

The configuration file uses YAML format with the following structure:

```yaml
packages:
  - name: <unique-identifier>
    display_name: <display-name>
    install_command: <installation-command>
    version_check_command: <latest-version-command>
    local_version_command: <installed-version-command>
```

### Field Descriptions

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique identifier for the package (used internally) |
| `display_name` | Yes | Human-readable name shown in the TUI |
| `install_command` | Yes | Command to install or update the package |
| `version_check_command` | Yes | Command to query the latest version from registry |
| `local_version_command` | Yes | Command to check the locally installed version |

### Configuration File Locations

The tool searches for `config.yaml` in the following order:

1. Current working directory: `./config.yaml`
2. User config directory: `~/.config/pnpm-manager/config.yaml`

### Example Configurations

#### TypeScript

```yaml
- name: typescript
  display_name: TypeScript
  install_command: npm install -g typescript@latest
  version_check_command: npm view typescript version
  local_version_command: tsc --version
```

#### Scoped Package

```yaml
- name: qwen-code
  display_name: Qwen Code CLI
  install_command: pnpm install -g @qwen-code/qwen-code@latest
  version_check_command: pnpm view @qwen-code/qwen-code@latest version
  local_version_command: qwen --version
```

#### Using pnpm

```yaml
- name: vite
  display_name: Vite
  install_command: pnpm install -g vite@latest
  version_check_command: pnpm view vite version
  local_version_command: vite --version
```

## Package Status

The tool automatically determines package status:

- **Not Installed**: Package is not found on the system
- **Installed**: Package is installed and up-to-date
- **Update Available**: A newer version is available
- **Checking...**: Currently fetching version information
- **Error**: Failed to check version or execute command

## Troubleshooting

### Config file not found

**Error**: `Failed to load config: config file not found`

**Solution**: Create a `config.yaml` file in your current directory or `~/.config/pnpm-manager/`. See the [Configuration](#configuration) section for the format.

### npm/pnpm not found

**Error**: `Command failed: npm not found` or similar

**Solution**: Ensure npm or pnpm is installed and available in your system PATH:

```bash
# Check if npm is installed
npm --version

# Check if pnpm is installed
pnpm --version
```

### Version parsing errors

**Error**: `Failed to parse version: no valid version found`

**Solution**: Verify that your `version_check_command` and `local_version_command` output valid semantic version numbers. The tool expects output containing versions in the format `X.Y.Z` (e.g., `1.2.3`).

Test your commands manually:

```bash
# Should output a version number
npm view typescript version
tsc --version
```

### Permission denied during installation

**Error**: `Installation failed: permission denied`

**Solution**: 
- On Unix/macOS: You may need to configure npm to install packages without sudo. See [npm documentation](https://docs.npmjs.com/resolving-eacces-permissions-errors-when-installing-packages-globally).
- On Windows: Run the terminal as Administrator.

### Network timeout

**Error**: `Command execution timeout`

**Solution**: 
- Check your internet connection
- Verify npm registry is accessible: `npm ping`
- The tool has a 30-second timeout for all commands. If your network is slow, commands may timeout.

### Terminal too small

**Issue**: UI appears garbled or incomplete

**Solution**: Resize your terminal window to at least 80x24 characters. The tool requires a minimum terminal size to display properly.

### Invalid YAML syntax

**Error**: `Failed to load config: yaml: line X: ...`

**Solution**: Check your `config.yaml` for syntax errors:
- Ensure proper indentation (use spaces, not tabs)
- Verify all strings are properly quoted if they contain special characters
- Check that all required fields are present

Use a YAML validator to check your file:

```bash
# Using Python
python -c "import yaml; yaml.safe_load(open('config.yaml'))"
```

## Architecture

The tool is built with a layered architecture:

- **Config Layer**: YAML parsing and validation
- **Utils Layer**: Cross-platform command execution and version comparison
- **Services Layer**: Package management business logic
- **UI Layer**: Bubble Tea TUI with Model-View-Update pattern

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./config
```

### Project Structure

```
my-pnpm-installer/
├── main.go              # Entry point
├── config/              # Configuration parsing
├── services/            # Package management logic
├── ui/                  # TUI implementation
├── utils/               # Command execution and version comparison
└── testdata/            # Test fixtures
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - see LICENSE file for details

## Acknowledgments

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [go-yaml](https://github.com/go-yaml/yaml) - YAML parsing
- [semver](https://github.com/Masterminds/semver) - Semantic versioning
