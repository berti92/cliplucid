# Cliplucid

Cliplucid is a macOS hotkey tool that sends clipboard and optional voice context to an OpenAI-compatible local API.

## Features

- Three action modes:
  - `popup`: uses `%CLIPBOARD%`, shows the answer as rendered Markdown in a browser popup
  - `popup_voice`: uses `%CLIPBOARD%` + `%VOICECONTEXT%`, shows the answer as rendered Markdown in a browser popup
  - `clipboard_voice`: uses `%CLIPBOARD%` + `%VOICECONTEXT%`, writes the answer to clipboard and optionally plays a completion sound
- Configurable global hotkeys and prompt files (YAML)
- Hold-to-record voice flow: recording starts on key down and stops on key up
- Works with a local OpenAI-compatible `v1/chat/completions` endpoint

## Requirements

- macOS (current version)
- Go 1.22+
- Accessibility permission for global hotkeys:
  - `System Settings -> Privacy & Security -> Accessibility`
- Microphone permission for audio capture:
  - `System Settings -> Privacy & Security -> Microphone`
- `sox` for recording (`brew install sox`)
- `whisper-cli` (from `whisper.cpp`) for local speech-to-text

## Installation

```bash
go mod tidy
go build -o cliplucid .
```

## Configuration

1. Copy the example config:

```bash
cp config.example.yaml config.yaml
```

2. Edit `config.yaml`:

- `api.base_url` (for example `http://localhost:1234/v1`)
- `api.model`
- `voice.language` (for example `de`, `en`, `fr`)
- `voice.whisper_model` (path to a local `ggml-*.bin` model file)
- `actions` (`hotkey`, `mode`, `prompt_file`, optional `done_sound`)

## Prompt Placeholders

- `%CLIPBOARD%`: current clipboard content
- `%VOICECONTEXT%`: speech-to-text result

Relative `voice.whisper_model` paths are resolved in this order:

1. Directory of the active `config.yaml`
2. `~/.config/cliplucid/`

Absolute `voice.whisper_model` paths are used as-is and must exist.

## Run

```bash
./cliplucid -config ./config.yaml
```

The process runs continuously and listens for configured hotkeys.

If `./config.yaml` is not present and you use the default config path, Cliplucid automatically falls back to:

```text
~/.config/cliplucid/config.yaml
```

Prompt resolution for relative `prompt_file` paths uses this order:

1. Directory of the active `config.yaml`
2. `~/.config/cliplucid/`

Absolute `prompt_file` paths are used as-is and must exist.

## Voice Transcription Notes

- Recording uses a built-in `sox` command (`sox -q -d <file> rate 16000 channels 1`) and is stopped with `SIGINT` on key release.
- After recording, Cliplucid runs `whisper-cli` internally using `voice.whisper_model` and `voice.language`.
- No external transcription scripts are required.

## Example whisper.cpp Setup

```bash
brew install whisper-cpp sox
export WHISPER_MODEL=/absolute/path/to/ggml-small.bin
```

Example config:

```yaml
voice:
  language: "de"
  whisper_model: "/absolute/path/to/ggml-small.bin"
```
