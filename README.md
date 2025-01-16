# MassRelay

## Overview

MassRelay is a Go-based tool for uploading multiple files to an HTTP server efficiently. It automatically uploads all files in the directory from which it's run. It includes a **Simulation Mode** for testing the robustness of its built-in retry policy by simulating server failures. This project is built using **Go version 1.23.4**.

## Features

- **Automatic Directory Upload**: Uploads all files in the directory where the tool is executed.
- **Concurrent HTTP Uploads**: Leverage Go's goroutines for parallel file uploading.
- **Built-in Retry Policy**: Automatically retry failed uploads with exponential backoff.
- **Simulation Mode**: Spins up a local server that randomly fails 10% of uploads to test the retry mechanism.
- **Configurable**: Uses a configuration file for settings.

## Installation

### Prerequisites

- **Go**: Version **1.23.4** or later
- **Git**: For cloning the repository

### Setup

1. **Clone the repository:**
   ```sh
   git clone https://github.com/justinwilliams-io/MassRelay.git
   cd MassRelay

2. **Build the project.**
    ```sh
    go build -o massrelay

## Usage

### Basic Upload

Run the tool from the directory containing the files you want to upload:

```sh
./massrelay

By default, it will look for configuration in $HOME/mass-relay/config.yaml.```

### Simulation Mode

To run in simulation mode, add the --simulate flag

```sh
./massrelay --simulate```

This will start a local server on port 8080 with a 10% failure rate for uploads.

### Options

- `--simulate`: Enable simulation mode for testing retry policy.

## Configuration

MassRelay uses a YAML configuration file located in the user's config directory:
- On **Unix-like systems (including Linux and macOS)**, this is typically `$XDG_CONFIG_HOME/mass-relay/config.yaml` or `~/.config/mass-relay/config.yaml` if `$XDG_CONFIG_HOME` is not set.
- On **Windows**, this would be `%APPDATA%\mass-relay\config.yaml`.

Here's an example of what the config might look like:

```yaml
remote_url: https://your-upload-endpoint
max_concurrent_uploads: 3
log_level: info  # Not implemented yet
token: your_api_token```


