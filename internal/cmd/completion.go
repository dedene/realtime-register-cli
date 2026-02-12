package cmd

import (
	"fmt"
	"os"
)

// CompletionCmd generates shell completions.
type CompletionCmd struct {
	Shell string `arg:"" help:"Shell type" enum:"bash,zsh,fish"`
}

func (c *CompletionCmd) Run(_ *RootFlags) error {
	switch c.Shell {
	case "bash":
		fmt.Fprintln(os.Stdout, bashCompletion)
	case "zsh":
		fmt.Fprintln(os.Stdout, zshCompletion)
	case "fish":
		fmt.Fprintln(os.Stdout, fishCompletion)
	}
	return nil
}

const bashCompletion = `_rr() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="version auth config status domain contact zone process tld completion --help --json --plain --verbose --yes --color --version"

    case "${prev}" in
        rr)
            COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
            return 0
            ;;
        domain)
            COMPREPLY=( $(compgen -W "list get check register update delete renew" -- ${cur}) )
            return 0
            ;;
        contact)
            COMPREPLY=( $(compgen -W "list get create update delete" -- ${cur}) )
            return 0
            ;;
        zone)
            COMPREPLY=( $(compgen -W "list get create update delete" -- ${cur}) )
            return 0
            ;;
        process)
            COMPREPLY=( $(compgen -W "list get info cancel resend" -- ${cur}) )
            return 0
            ;;
        tld)
            COMPREPLY=( $(compgen -W "list get" -- ${cur}) )
            return 0
            ;;
        auth)
            COMPREPLY=( $(compgen -W "login status logout" -- ${cur}) )
            return 0
            ;;
        config)
            COMPREPLY=( $(compgen -W "get set list path" -- ${cur}) )
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
            return 0
            ;;
    esac

    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
}
complete -F _rr rr`

const zshCompletion = `#compdef rr

_rr() {
    local -a commands
    commands=(
        'version:Show version information'
        'auth:Manage API key'
        'config:Manage configuration'
        'status:Show account status'
        'domain:Domain commands'
        'contact:Contact commands'
        'zone:DNS zone commands'
        'process:Process commands'
        'tld:TLD commands'
        'completion:Generate shell completions'
    )

    _arguments -C \
        '(-h --help)'{-h,--help}'[Show help]' \
        '(-j --json)'{-j,--json}'[Output JSON]' \
        '--plain[Output TSV]' \
        '(-v --verbose)'{-v,--verbose}'[HTTP debug logging]' \
        '(-y --yes)'{-y,--yes}'[Skip confirmations]' \
        '--color[Color mode]:mode:(auto always never)' \
        '--version[Print version]' \
        '1: :->command' \
        '*::arg:->args'

    case $state in
        command)
            _describe -t commands 'rr command' commands
            ;;
    esac
}

_rr "$@"`

const fishCompletion = `complete -c rr -f

complete -c rr -n "__fish_use_subcommand" -a version -d "Show version"
complete -c rr -n "__fish_use_subcommand" -a auth -d "Manage API key"
complete -c rr -n "__fish_use_subcommand" -a config -d "Manage configuration"
complete -c rr -n "__fish_use_subcommand" -a status -d "Show account status"
complete -c rr -n "__fish_use_subcommand" -a domain -d "Domain commands"
complete -c rr -n "__fish_use_subcommand" -a contact -d "Contact commands"
complete -c rr -n "__fish_use_subcommand" -a zone -d "DNS zone commands"
complete -c rr -n "__fish_use_subcommand" -a process -d "Process commands"
complete -c rr -n "__fish_use_subcommand" -a tld -d "TLD commands"
complete -c rr -n "__fish_use_subcommand" -a completion -d "Generate completions"

complete -c rr -n "__fish_seen_subcommand_from domain" -a "list get check register update delete renew"
complete -c rr -n "__fish_seen_subcommand_from contact" -a "list get create update delete"
complete -c rr -n "__fish_seen_subcommand_from zone" -a "list get create update delete"
complete -c rr -n "__fish_seen_subcommand_from process" -a "list get info cancel resend"
complete -c rr -n "__fish_seen_subcommand_from tld" -a "list get"
complete -c rr -n "__fish_seen_subcommand_from auth" -a "login status logout"
complete -c rr -n "__fish_seen_subcommand_from config" -a "get set list path"
complete -c rr -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"

complete -c rr -s h -l help -d "Show help"
complete -c rr -s j -l json -d "Output JSON"
complete -c rr -l plain -d "Output TSV"
complete -c rr -s v -l verbose -d "HTTP debug"
complete -c rr -s y -l yes -d "Skip confirmations"
complete -c rr -l color -xa "auto always never" -d "Color mode"`
