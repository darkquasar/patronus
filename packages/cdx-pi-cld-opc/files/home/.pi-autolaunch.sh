# Sourced from ~/.bashrc. Defaults `pi` to brokered OpenAI and starts the first
# interactive shell in pi. Exit pi (Ctrl+D or /exit) for a shell.

# Bare `pi` gets the brokered provider; --provider or --model overrides it, and
# `command pi` bypasses this function.
pi() {
  case " $* " in
    *" --provider "*|*" --model "*) command pi "$@" ;;
    *) command pi --provider openai-codex --model gpt-6.1-sol "$@" ;;
  esac
}

case $- in
  *i*)
    if [ -z "${PI_SHELL_STARTED:-}" ]; then
      export PI_SHELL_STARTED=1
      printf '\n-> pi (openai-codex / gpt-6.1-sol). Exit pi for a shell. See ~/workspace/QUICKSTART.md\n\n'
      pi
    fi
    ;;
esac
