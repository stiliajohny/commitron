[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![GPL3 License][license-shield]][license-url]
[![LinkedIn][linkedin-shield]][linkedin-url]
[![Ask Me Anything][ask-me-anything]][personal-page]

<!-- PROJECT LOGO -->
<br />
<p align="center">
  <a href="https://github.com/stiliajohny/commitron">
    <img src="https://raw.githubusercontent.com/stiliajohny/commitron/main/.assets/logo.png" alt="Main Logo" width="80" height="80">
  </a>

  <h3 align="center">commitron</h3>

  <p align="center">
    AI-driven CLI tool that generates optimal, context-aware commit messages, streamlining your version control process with precision and efficiency
    <br />
    <a href="./README.md"><strong>Explore the docs »</strong></a>
    <br />
    <br />
    <a href="https://github.com/stiliajohny/commitron/issues/new?labels=i%3A+bug&template=1-bug-report.md">Report Bug</a>
    ·
    <a href="https://github.com/stiliajohny/commitron/issues/new?labels=i%3A+enhancement&template=2-feature-request.md">Request Feature</a>
  </p>
</p>

<!-- TABLE OF CONTENTS -->

## Table of Contents

- [Commitron](#commitron)
  - [Features](#features)
  - [Example output](#example-output)
  - [Installation](#installation)
  - [Usage](#usage)
  - [Configuration](#configuration)
  - [API Keys](#api-keys)
  - [License](#license)
    - [Built With](#built-with)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Installation](#installation-1)
  - [Usage](#usage-1)
  - [Roadmap](#roadmap)
  - [Contributing](#contributing)
  - [License](#license-1)
  - [Contact](#contact)
  - [Acknowledgements](#acknowledgements)

<!-- ABOUT THE PROJECT -->

## About The Project

<!-- [![commitron Screen Shot][product-screenshot]](./.assets/screenshot.png) -->

# Commitron

Commitron is a CLI tool that generates AI-powered commit messages based on your staged changes in a git repository.

## Features

- 🤖 Uses AI to generate meaningful commit messages
- 🔍 Analyzes your staged changes to understand what has been modified
- 🧩 Supports multiple AI providers:
  - OpenAI (ChatGPT)
  - Google Gemini
  - Ollama (local inference)
  - Anthropic Claude
- 📝 Supports various commit message conventions:
  - No convention (plain text)
  - [Conventional Commits](https://www.conventionalcommits.org/)
  - Custom templates
- ⚙️ Customizable settings via configuration file

## Example output

When you run `commitron` after staging some changes, you'll see output similar to this:

```
Generated Commit Message:
------------------------
feat(git): add support for detecting staged files and creating commits

Implement Git integration for detecting staged files and generating
commit messages based on the changes. This adds functionality to check
if the current directory is a Git repository, retrieve a list of staged
files, and create commits with AI-generated messages.
------------------------

Do you want to use this commit message? (y/n):
```

If you type `y`, the commit will be created using the generated message.

## Installation

### Using Homebrew (macOS)

```bash
# Add the tap first
brew tap stiliajohny/tap https://github.com/stiliajohny/homebrew-tap.git

# Then install commitron
brew install commitron
```

### Manual Installation

```bash
# Clone the repository
git clone https://github.com/johnstilia/commitron.git

# Navigate to the directory
cd commitron

# Build and install
go install ./cmd/commitron
```

## Usage

```bash
# Stage your changes
git add .

# Run commitron to generate a commit message
commitron

# Use with a custom config file
commitron --config /path/to/custom/config.yaml
# or using shorthand flags
commitron -c /path/to/custom/config.yaml

# Available commands
commitron generate            # Generate a commit message (default command)
commitron init                # Initialize a new configuration file
commitron version             # Show version information

# Command options
commitron generate --dry-run                  # Preview without committing
commitron generate -d                         # Shorthand for --dry-run
commitron generate -c /path/to/config         # Use custom config (shorthand)
commitron init --force                        # Overwrite existing config
commitron init -f                             # Shorthand for --force
commitron init -c /path/to/config             # Initialize at custom location (shorthand)

# Get help for any command
commitron --help
commitron [command] --help
```

## Configuration

Commitron looks for a configuration file at `~/.commitronrc`. This is a YAML file that allows you to customize how the tool works.

Example configuration:

```yaml
# AI provider configuration
ai:
  provider: openai
  api_key: your-api-key-here
  model: gpt-3.5-turbo

# Commit message configuration
commit:
  convention: conventional
  include_body: true
  max_length: 72

# Context settings for AI
context:
  include_file_names: true
  include_diff: true
  max_context_length: 4000
```

See [example.commitronrc](example.commitronrc) for a complete example with all available options.

## API Keys

To use Commitron, you'll need API keys for your chosen AI provider:

- OpenAI: <https://platform.openai.com/api-keys>
- Google Gemini: <https://aistudio.google.com/app/apikey>
- Anthropic Claude: <https://console.anthropic.com/keys>

For Ollama, you need to have it running locally. See [Ollama documentation](https://github.com/ollama/ollama) for more information.

## License

See [LICENSE.txt](LICENSE.txt) for details.

### Built With

<!--
This section should list any major frameworks that you built your project using. Leave any add-ons/plugins for the acknowledgements section. Here are a few examples.

- [Bootstrap](https://getbootstrap.com)
- [JQuery](https://jquery.com)
- [Laravel](https://laravel.com)
-->

---

<!-- GETTING STARTED -->

## Getting Started

<!--
This is an example of how you may give instructions on setting up your project locally.
To get a local copy up and running follow these simple example steps.
-->

### Prerequisites

<!--

This is an example of how to list things you need to use the software and how to install them.

- npm

```sh
npm install npm@latest -g
```
-->

### Installation

<!--
1. Get a free API Key at [https://example.com](https://example.com)
2. Clone the repo

```sh
git clone https://github.com/your_username_/Project-Name.git
```

3. Install NPM packages

```sh
npm install
```

4. Enter your API in `config.js`

```JS
const API_KEY = 'ENTER YOUR API';
```
-->

---

<!-- USAGE EXAMPLES -->

## Usage

<!--
Use this space to show useful examples of how a project can be used. Additional screenshots, code examples and demos work well in this space. You may also link to more resources.

_For more examples, please refer to the [Documentation](https://example.com)_
-->

---

<!-- ROADMAP -->

## Roadmap

See the [open issues](https://github.com/stiliajohny/commitron/issues) for a list of proposed features (and known issues).

---

<!-- CONTRIBUTING -->

## Contributing

Contributions are what make the open source community such an amazing place to be learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

<!-- LICENSE -->

## License

Distributed under the GPLv3 License. See `LICENSE` for more information.

<!-- CONTACT -->

## Contact

John Stilia - <stilia.johny@gmail.com>

<!--
Project Link: [https://github.com/your_username/repo_name](https://github.com/your_username/repo_name)
-->

---

<!-- ACKNOWLEDGEMENTS -->

## Acknowledgements

- [GitHub Emoji Cheat Sheet](https://www.webpagefx.com/tools/emoji-cheat-sheet)
- [Img Shields](https://shields.io)
- [Choose an Open Source License](https://choosealicense.com)
- [GitHub Pages](https://pages.github.com)

<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->

[contributors-shield]: https://img.shields.io/github/contributors/stiliajohny/commitron.svg?style=for-the-badge
[contributors-url]: https://github.com/stiliajohny/commitron/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/stiliajohny/commitron.svg?style=for-the-badge
[forks-url]: https://github.com/stiliajohny/commitron/network/members
[stars-shield]: https://img.shields.io/github/stars/stiliajohny/commitron.svg?style=for-the-badge
[stars-url]: https://github.com/stiliajohny/commitron/stargazers
[issues-shield]: https://img.shields.io/github/issues/stiliajohny/commitron.svg?style=for-the-badge
[issues-url]: https://github.com/stiliajohny/commitron/issues
[license-shield]: https://img.shields.io/github/license/stiliajohny/commitron?style=for-the-badge
[license-url]: https://github.com/stiliajohny/commitron/blob/master/LICENSE.txt
[linkedin-shield]: https://img.shields.io/badge/-LinkedIn-black.svg?style=for-the-badge&logo=linkedin&colorB=555
[linkedin-url]: https://linkedin.com/in/johnstilia/
[ask-me-anything]: https://img.shields.io/badge/Ask%20me-anything-1abc9c.svg?style=for-the-badge
[personal-page]: https://github.com/stiliajohny
