# AI Git Commit Message Generator

A command-line tool that leverages AI to generate meaningful git commit messages based on your changes.

## Features

- 🤖 AI-powered commit message generation
- 🚀 Fast and lightweight
- 🔌 Supports multiple AI providers
- 💻 Easy-to-use CLI interface
- ⚙️ Configurable output format

## Installation

### Quick Install (Linux and macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/edhuardotierrez/gommit/main/install.sh | bash
```

### Manual Installation

```bash
go install github.com/edhuardotierrez/gommit@latest
```

## Configuration

You can configure gommit in two ways:

### 1. Using the Configuration Wizard

Run the configuration wizard:
```bash
gommit --config
```

The wizard will guide you through setting up:
- AI Provider (e.g., OpenAI)
- API Key
- Model selection
- Maximum tokens for responses
- Commit message style

### 2. Manual Configuration

Create a configuration file at `~/gommit.json`:

```json
{
  "default_provider": "openai",
  "providers": {
    "openai": {
      "api_key": "your-api-key-here",
      "model": "gpt-3.5-turbo"
    }
  },
  "max_tokens": 500,
  "commit_style": "conventional"
}
```

### Configuration Options

| Option | Description | Example Values |
|--------|-------------|----------------|
| `default_provider` | The AI provider to use | `"openai"`, `"anthropic"` |
| `api_key` | Your API key for the provider | `"sk-..."` |
| `model` | The model to use | `"gpt-3.5-turbo"`, `"gpt-4"` |
| `max_tokens` | Maximum tokens in the response | `500`, `1000` |
| `commit_style` | Style of commit messages | `"conventional"`, `"descriptive"` |

## Usage

1. Stage your changes using `git add`
2. Run `gommit` in your git repository
3. The tool will analyze your changes and generate a commit message
4. The commit will be created automatically

## Features

- Generates meaningful commit messages based on staged changes
- Supports multiple LLM providers through [lingoose](https://github.com/henomis/lingoose)
- Reads configuration from user's home directory
- Falls back to standard git commit output on errors

## License

MIT License - See LICENSE file for details
