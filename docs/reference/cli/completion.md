---
title: gh app-auth completion
description: Generate the autocompletion script for the specified shell
---

This page is generated from the output of `gh-app-auth completion --help` in the checked-out source. To refresh it, run `npm run reference:generate` in `website/`.

> Help text and generated scripts invoke the standalone `gh-app-auth` command, so it must be an executable on `PATH`. An extension-only install may not provide it. The generator can be invoked as `gh app-auth completion <shell>`, but the emitted scripts still target the standalone command. See [Install](../../start-here/first-clone.md#install) and [development](../../contribute/development.md). This does not add `gh app-auth` completion.

## Help output

```text
Generate the autocompletion script for gh-app-auth for the specified shell.
See each sub-command's help for details on how to use the generated script.

Usage:
  gh-app-auth completion [command]

Available Commands:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh

Flags:
  -h, --help   help for completion

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output

Use "gh-app-auth completion [command] --help" for more information about a command.
```

## gh app-auth completion bash

```text
Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

  source <(gh-app-auth completion bash)

To load completions for every new session, execute once:

#### Linux:

  gh-app-auth completion bash > /etc/bash_completion.d/gh-app-auth

#### macOS:

  gh-app-auth completion bash > $(brew --prefix)/etc/bash_completion.d/gh-app-auth

You will need to start a new shell for this setup to take effect.

Usage:
  gh-app-auth completion bash

Flags:
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## gh app-auth completion fish

```text
Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

  gh-app-auth completion fish | source

To load completions for every new session, execute once:

  gh-app-auth completion fish > ~/.config/fish/completions/gh-app-auth.fish

You will need to start a new shell for this setup to take effect.

Usage:
  gh-app-auth completion fish [flags]

Flags:
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## gh app-auth completion powershell

```text
Generate the autocompletion script for powershell.

To load completions in your current shell session:

  gh-app-auth completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.

Usage:
  gh-app-auth completion powershell [flags]

Flags:
  -h, --help              help for powershell
      --no-descriptions   disable completion descriptions

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## gh app-auth completion zsh

```text
Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

  echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

  source <(gh-app-auth completion zsh)

To load completions for every new session, execute once:

#### Linux:

  gh-app-auth completion zsh > "${fpath[1]}/_gh-app-auth"

#### macOS:

  gh-app-auth completion zsh > $(brew --prefix)/share/zsh/site-functions/_gh-app-auth

You will need to start a new shell for this setup to take effect.

Usage:
  gh-app-auth completion zsh [flags]

Flags:
  -h, --help              help for zsh
      --no-descriptions   disable completion descriptions

Global Flags:
      --config string   Path to configuration file
      --debug           Enable debug output
```

## Related

- [Install](../../start-here/first-clone.md#install)
