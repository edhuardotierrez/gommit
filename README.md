# (go)mmit - Automated Git Commit Messages with LLM 🤖 written in Go

## Features

- 🤖 AI-powered git commit message generator based on your staged changes.
- 🚀 Fast and lightweight
- 🔌 Supports multiple AI providers
- 💻 Easy-to-use CLI interface
- ⚙️ Configurable output format

## Installation

### Quick Install (Linux and macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/edhuardotierrez/gommit/main/install.sh | bash
```

### Build and install manually (Any OS)

Required **Go 1.25+** ([https://go.dev/dl/](https://go.dev/dl/))

```bash
go install github.com/edhuardotierrez/gommit@latest
```

## Configuration

You can configure gommit in two ways:

### 1. Using the Configuration Wizard

Run the configuration wizard:

```bash
gommit -config
```

The wizard will guide you through setting up:

- AI Provider (e.g., OpenAI)
- API Key
- Model selection
- Maximum tokens for responses
- Commit message style

### 2. Manual Configuration

Create (or edit) a configuration file at `~/gommit.json`:

```json
{
  "default_provider": "openai",
  "providers": {
    "openai": {
      "api_key": "your-api-key-here",
      "model": "gpt-4o-mini",
      "temperature": 0.5
    },
    "ollama": {
      "uri": "http://localhost:11434",
      "model": "llama3.1:8b",
      "temperature": 0.5
    }
  },
  "max_tokens": 500,
  "commit_style": "conventional"
}
```

### Configuration Options

| Option             | Description                                          | Example Values                             |
| ------------------ | ---------------------------------------------------- | ------------------------------------------ |
| `default_provider` | The AI provider to use                               | `"openai"`, `"anthropic"`                  |
| `api_key`          | Your API key for the provider                        | `"sk-..."`                                 |
| `model`            | The model to use                                     | `"gpt-4o-mini"`, `"gpt-4"`                 |
| `max_tokens`       | Maximum tokens in the response                       | `500`, `1000`                              |
| `commit_style`     | Style of commit messages                             | `"conventional"`, `"simple"`, `"detailed"` |
| `temperature`      | Temperature for the response                         | `0.5`, `1.0`                               |
| `uri`              | The URI of the provider                              | `"http://localhost:11434"`                 |
| `truncate_lines`   | Number of context lines to include in each file diff | `3`, `5`, `10`                             |
| `max_line_width`   | Maximum line width in each file diff                 | `120`, `100`, `80`                         |

Note: The default values are `1000` for `truncate_lines` and `300` for `max_line_width`.

### Commit Style

The commit style is the style of the commit message. The default style is `conventional`.
The available styles are:

- `conventional`: Add a conventional style commit message, using more general, flexible and readablemessage, use context and more information about the changes (less than 500 characters).
- `simple`: Add a simple and short commit message, reducing the amount of information to a minimum (less than 100 characters).
- `detailed`: Add a detailed commit message, with more information about the changes, variables names, context and files affected (less than 1000 characters).

## Custom Commit Rules

You can customize the commit message generation rules by creating a `.gommitrules` file in your repository's root directory. This file is optional, and if it is not present, gommit will use the default rules based on your Commit Style.

Note: You are responsible for ensuring commit messages follow these rules.

## Usage

System requirements:

- Git installed (1.8.5+) [https://git-scm.com/downloads]
- AI provider API key (e.g. OpenAI, Anthropic, Ollama, etc.)

Steps:

1. Stage your changes using `git add <file> <file> ...`
2. Run `gommit` command in your git repository, it will analyze your changes and generate a commit message
3. Preview the commit message and confirm it, if you are happy with the message, it will be created automatically (`git commit -m "<generated commit message>"`)

## Override configuration options

These command line flags will not affect your configuration file (`~/gommit.json`):

```bash
gommit -p <provider> -m <model> -t <temperature> -s <style>
```

Examples:

```bash
# Use OpenAI's GPT-4o-mini model with custom settings
gommit -p openai -m gpt-4o-mini -t 0.5 -s detailed

# Use a simple short commit message
gommit -s simple

# Use Anthropic's Claude model with high temperature
gommit -p anthropic -m claude-3-5-sonnet-latest -t 0.8

# Use a specific truncate lines and max line width
gommit -l 3 -w 120

```

Note: Before using gommit, you'll need to configure your providers, models, and API keys. You can do this by either:

- Running the configuration wizard with `gommit -config`
- Manually editing the configuration file at `~/gommit.json`

## Todo:

- [x] Add support for main AI providers:
  - [x] OpenAI
  - [x] Anthropic
  - [x] Ollama
  - [ ] Gemini
- [x] Support for configuration wizard (`gommit -config`)
- [x] Support for persistent configuration file (`~/gommit.json`)
- [x] Add support for override configuration options with flags
- [x] Add support for custom commit rules per repository (`.gommitrules`)
- [x] Add support for truncate lines and max line width in the diff

## License

MIT License - See LICENSE file for details
