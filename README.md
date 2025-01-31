# AI Git Commit Message Generator

A command-line tool that leverages AI to generate meaningful git commit messages based on your changes.

## Features

- 🤖 AI-powered commit message generation
- 🚀 Fast and lightweight
- 🔌 Supports multiple AI providers
- 💻 Easy-to-use CLI interface
- ⚙️ Configurable output format

## Installation

```bash
go install github.com/edhuardotierrez/gommit@latest
```

## Configuration

Create a configuration file at `~/gommit.json` with your LLM API keys:

```json
{
    "llm_provider": "openai",
    "api_key": "your-api-key-here"
}
```

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
