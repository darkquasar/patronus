# Quickstart

`sbx exec -it <name> bash` starts in **pi** on brokered OpenAI. Exit pi
(Ctrl+D or `/exit`) for a shell; the session stays open.

| want | command |
|------|---------|
| pi, brokered OpenAI | `pi` (openai-codex / gpt-6.1-sol) |
| pi, astra | `pi --provider openai-codex --model gpt-6-astra` |
| pi on Anthropic | `pi --provider anthropic`, then `/login` |
| opencode, brokered OpenAI | `opencode run "..."` (gpt-6.1-sol) |
| opencode, astra | `opencode run -m openai/gpt-6-astra "..."` |
| codex | `codex exec "..."` |
| claude | `claude`, then `/login` |

Bare `pi` is a shell function that adds `--provider openai-codex --model
gpt-6.1-sol`. Pass `--provider` or `--model` to override, or run `command pi` to
bypass it. Brokered models are `gpt-6.1-sol` and `gpt-6-astra`, with no key in
the VM. Anthropic needs its own login.
